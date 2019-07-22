package slack

import (
	"fmt"
	"github.com/miguelff/cable/cable"
	"github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
	"strings"
	"sync"
	"time"
)

/* Section: Slack API interface and its slack.Client adapter */

// UserMap is a collection of slack Users indexed by their ID, which is a string
type UserMap map[string]slack.User

// API lets us replace the slack API Client with something that behaves like it.
// This is used to improve testability
type API interface {
	// IncomingEvents returns the channel of events slack is listening to
	IncomingEvents() <-chan slack.RTMEvent
	// PostMessage Posts a message in a slack channel
	PostMessage(channelID string, options ...slack.MsgOption) error
	// GetUsers retrieves information about the Users in the slack workspace the
	// app is connected to
	GetUsers() UserMap
}

// APIAdapter Adapts an api.Client to conform to the API interface
type APIAdapter struct {
	// Client is the adapted Client
	Client *slack.Client
	// RTMEvents is a local reference to the channels of events coming from slack
	RTMEvents chan slack.RTMEvent
	// userIdentitiesCache is a local cache of the list of Users
	// in the workspace slack is installed in
	userIdentitiesCache UserMap
	// cache mutex controls access to the cache by multiple goroutines
	cacheMutex sync.Mutex
}

// IncomingEvents returns the channel of RTMEvents managed by the slack's API
// Client.
//
// When called for the first time, it lazily spawns a goroutine to manage the
// connection to the RTM api, caching locally a reference to the channel of
// updates.
func (adapter *APIAdapter) IncomingEvents() <-chan slack.RTMEvent {
	if adapter.RTMEvents == nil {
		rtm := adapter.Client.NewRTM()
		go rtm.ManageConnection()
		adapter.RTMEvents = rtm.IncomingEvents
	}
	return adapter.RTMEvents
}

// PostMessage forwards the call to the adapted Client's PostMessage method
func (adapter *APIAdapter) PostMessage(channelID string, options ...slack.MsgOption) error {
	_, _, err := adapter.Client.PostMessage(channelID, options...)
	return err
}

// GetUsers returns the user information from slack and caches it locally for
// a minute
func (adapter *APIAdapter) GetUsers() UserMap {
	adapter.cacheMutex.Lock()
	if adapter.userIdentitiesCache != nil {
		defer adapter.cacheMutex.Unlock()
		return adapter.userIdentitiesCache
	}
	adapter.cacheMutex.Unlock()

	// clear the cache every 1 minute
	defer func() {
		go func() {
			<-time.NewTimer(1 * time.Minute).C
			log.Debugln("Clearing Users cache...")
			adapter.cacheMutex.Lock()
			adapter.userIdentitiesCache = nil
			adapter.cacheMutex.Unlock()
		}()
	}()

	users, err := adapter.Client.GetUsers()
	if err != nil {
		log.Errorln("Cannot get user identities", err)
	}
	res := make(UserMap)
	for _, u := range users {
		res[u.ID] = u
	}

	if err == nil {
		log.Debugln("Setting Users cache...")
		adapter.cacheMutex.Lock()
		adapter.userIdentitiesCache = res
		adapter.cacheMutex.Unlock()
	}

	return res
}

/* Section: Slack type implementing GoRead() and GoWrite() */

// Slack adapts the Telegram API creating a Pump of messages
type Slack struct {
	// Pump is the pair of InboxCh and OutboxCh channel to receive
	// messages from and write messages to Slack
	*cable.Pump
	// Client is the slack api Client
	client API
	// relayedChannelID is the ID of the channel messages will be read from and
	// relayed to
	relayedChannelID string
	// botUserID is the id of the slack installed in the organization, which is
	// used to discard messages looped back by the own bot
	botUserID string
}

// NewSlack returns the address of a new value of Slack
func NewSlack(token string, relayedChannel string, botUserID string) *Slack {
	return &Slack{
		Pump:             cable.NewPump(),
		client:           &APIAdapter{Client: slack.New(token)},
		relayedChannelID: relayedChannel,
		botUserID:        botUserID,
	}
}

// GetIdentities returns the user information from slack and caches it locally
// for a minute
func (s *Slack) GetIdentities() UserMap {
	return s.client.GetUsers()
}

// GoRead makes slack listening for messages in a different goroutine.
// Those messages will be pushed to the InboxCh of the Pump.
//
// The goroutine can be stopped by feeding ReadStopper synchronization channel
// which can be done by calling StopRead() - a method coming from Pump and
// which is accessed directly through the Slack value.
func (s *Slack) GoRead() {
	go func() {
		for {
			select {
			case msg := <-s.client.IncomingEvents():
				update, err := s.ToInboxUpdate(msg)
				if err != nil {
					log.Debugf("Update from inbox discarded: %s", err)
				} else {
					s.Inbox() <- update
				}
			case <-s.ReadStopper:
				return
			}
		}
	}()
}

// ToInboxUpdate converts any event received in the read pump to a cable update
// that can be fed into the inbox
func (s *Slack) ToInboxUpdate(update interface{}) (cable.Update, error) {
	switch ev := update.(slack.RTMEvent).Data.(type) {
	case *slack.MessageEvent:
		return s.toInboxMessage(ev)
	default:
		return nil, fmt.Errorf("ignoring unknown update type %s", update)
	}
}

func (s *Slack) toInboxMessage(msg *slack.MessageEvent) (cable.Update, error) {
	if msg.Channel != s.relayedChannelID || msg.BotID == s.botUserID {
		return nil, fmt.Errorf("ignoring message %s", msg.Text)
	}
	var author cable.Author

	users := s.GetIdentities()
	if user, ok := users[msg.User]; ok {
		author = cable.Author{
			Name:  user.RealName,
			Alias: user.Name,
		}
	} else {
		author = cable.Author{
			Name:  "Stranger",
			Alias: msg.User,
		}
	}

	return cable.Message{
		Author:   &author,
		Contents: &cable.Contents{msg.Text},
	}, nil
}

// GoWrite spawns a goroutine that takes care of delivering to slack the
// messages arriving at the OutboxCh of the Pump.
//
// The goroutine can be stopped by feeding WriteStopper synchronization channel
// which can be done by calling StopWrite() - a method coming from Pump and
// which is accessed directly through the Slack value.
func (s *Slack) GoWrite() {
	go func() {
		for {
			select {
			case ou := <-s.Outbox():
				update, err := s.FromOutboxUpdate(ou)
				if err != nil {
					log.Infof("Update from inbox discarded: %s", err)
				}
				err = s.Send(update)
				if err != nil {
					log.Errorln("Error sending message: ", err)
				}
			case <-s.WriteStopper:
				return
			}
		}
	}()
}

// Send sends a slack update
func (s *Slack) Send(update interface{}) error {
	err := s.client.PostMessage(s.relayedChannelID, update.([]slack.MsgOption)...)
	return err
}

// FromOutboxUpdate converts the given Update into a []slack.MsgOption message, which
// this pumper know how to send over the write
func (s *Slack) FromOutboxUpdate(update cable.Update) (interface{}, error) {
	switch ir := update.(type) {
	case cable.Message:
		return s.fromOutboxMessage(ir), nil
	default:
		return nil, fmt.Errorf("Cannot convert update to slack %s", update)
	}
}

func (s *Slack) fromOutboxMessage(m cable.Message) []slack.MsgOption {
	var author string
	var authorTokens []string

	if author := m.Author; author != nil {
		if author.Name != "" {
			authorTokens = append(authorTokens, author.Name)
		}
		if author.Alias != "" {
			if len(authorTokens) > 0 {
				authorTokens = append(authorTokens, fmt.Sprintf("(%s)", author.Alias))
			} else {
				authorTokens = append(authorTokens, author.Alias)
			}
		}
	}

	if len(authorTokens) > 0 {
		author = strings.Join(authorTokens, " ")
	} else {
		author = ""
	}

	attachment := slack.Attachment{
		Fallback:   m.Contents.String(),
		AuthorName: author,
		Text:       m.Contents.String(),
	}
	return []slack.MsgOption{slack.MsgOptionAttachments(attachment)}
}

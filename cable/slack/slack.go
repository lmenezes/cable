package slack

import (
	"fmt"
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/kyokomi/emoji"
	"github.com/miguelff/cable/cable"
	"github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
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
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
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
func (adapter *APIAdapter) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
	return adapter.Client.PostMessage(channelID, options...)
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
				switch ev := msg.Data.(type) {
				case *slack.MessageEvent:
					if ev.Channel != s.relayedChannelID || ev.BotID == s.botUserID {
						continue
					}
					s.Inbox() <- &Message{ev, s.GetIdentities()}
				}
			case <-s.ReadStopper:
				return
			}
		}
	}()
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
			case msg := <-s.Outbox():
				msgOptions, err := msg.ToSlack()
				if err != nil {
					log.Errorln("Slack error converting message to Client representation: ", err)
				}
				_, _, err = s.client.PostMessage(s.relayedChannelID, msgOptions...)
				if err != nil {
					log.Errorln("Slack error writing message: ", err)
				}
			case <-s.WriteStopper:
				return
			}
		}
	}()
}

/* Section: Slack message */

// Message wraps a message event from slack and implements the Message
// Interface
type Message struct {
	*slack.MessageEvent
	Users UserMap
}

// ToSlack is a no-op that returns an error, as we don't want to re-send
// messages from slack to slack at the moment.
func (sm Message) ToSlack() ([]slack.MsgOption, error) {
	return nil, fmt.Errorf("Messages received in slack are not sent to slack")
}

// ToTelegram converts a received slack message into a proper representation in
// telegram
func (sm Message) ToTelegram(telegramChatID int64) (telegram.MessageConfig, error) {
	var text string

	if user, ok := sm.Users[sm.User]; ok {
		text = fmt.Sprintf("*%s (%s):* %s", user.RealName, user.Name, sm.Text)
	} else {
		text = fmt.Sprintf("*Stranger:* %s", sm.Text)
	}

	return telegram.MessageConfig{
		BaseChat: telegram.BaseChat{
			ChatID:           telegramChatID,
			ReplyToMessageID: 0,
		},
		Text:                  emoji.Sprint(text),
		DisableWebPagePreview: false,
		ParseMode:             telegram.ModeMarkdown,
	}, nil
}

// String returns a human readable representation of a slack message for
// debugging purposes
func (sm Message) String() string {
	if user, ok := sm.Users[sm.User]; ok {
		return fmt.Sprintf("%s: %s", user.Name, sm.Text)

	}
	return fmt.Sprintf("Stranger: %s", sm.Text)
}

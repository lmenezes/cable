package cable

import (
	"fmt"
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/kyokomi/emoji"
	"github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

/* Section: Slack API interface and its slack.Client adapter */

// SlackAPI lets us replace the slack API client with something that behaves like it.
// This is used to improve testability
type SlackAPI interface {
	// IncomingEvents returns the channel of events the slack is listening to
	IncomingEvents() <-chan slack.RTMEvent
	// PostMessage Posts a message in a slack channel
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
	// GetUsers retrieves information about a workspace in Slack
	GetUsers() ([]slack.User, error)
}

// Adapts an api.Client to conform to the SlackAPI interface
type slackAPIAdapter struct {
	client    *slack.Client
	rtmEvents chan slack.RTMEvent
}

func (adapter *slackAPIAdapter) IncomingEvents() <-chan slack.RTMEvent {
	if adapter.rtmEvents == nil {
		rtm := adapter.client.NewRTM()
		go rtm.ManageConnection()
		adapter.rtmEvents = rtm.IncomingEvents
	}
	return adapter.rtmEvents
}

func (adapter *slackAPIAdapter) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
	return adapter.client.PostMessage(channelID, options...)
}

func (adapter *slackAPIAdapter) GetUsers() ([]slack.User, error) {
	return adapter.client.GetUsers()
}

// UserMap is a collection of slack Users indexed by their ID, which is a string
type UserMap map[string]slack.User

/* Section: Slack type implementing GoRead() and GoWrite() */

// Slack adapts the Telegram Api creating a Pump of messages
type Slack struct {
	// Pump is the pair of InboxCh and OutboxCh channel to receive
	// messages from and write messages to Slack
	*Pump
	// client is the slack api client
	client SlackAPI
	// relayedChannelID is the ID of the channel messages will be read from and
	// relayed to
	relayedChannelID string
	// botUserID is the id of the slack installed in the organization, which is
	// used to discard messages posted in slack as a result of relaying another
	// service
	botUserID string
	// userIdentitiesCache is a local cache of the list of users
	// in the workspace slack is installed in
	userIdentitiesCache UserMap
	// cache mutex controls access to the cache by multiple goroutines
	cacheMutex sync.Mutex
}

// NewSlack returns the address of a new value of Slack
func NewSlack(token string, relayedChannel string, botUserID string) *Slack {
	slack := &Slack{
		Pump:             NewPump(),
		client:           &slackAPIAdapter{client: slack.New(token)},
		relayedChannelID: relayedChannel,
		botUserID:        botUserID,
	}

	return slack
}

// GetIdentities returns the list of user information and caches it locally for
// a minute
func (s *Slack) GetIdentities() UserMap {
	s.cacheMutex.Lock()
	if s.userIdentitiesCache != nil {
		defer s.cacheMutex.Unlock()
		return s.userIdentitiesCache
	}
	s.cacheMutex.Unlock()

	// clear the cache every 1 minute
	defer func() {
		go func() {
			<-time.NewTimer(1 * time.Minute).C
			log.Debugln("Clearing users cache...")
			s.cacheMutex.Lock()
			s.userIdentitiesCache = nil
			s.cacheMutex.Unlock()
		}()
	}()

	log.Debugln("Setting users cache...")
	users, err := s.client.GetUsers()
	if err != nil {
		log.Errorln("Cannot get user identities", err)
	}
	res := make(UserMap)
	for _, u := range users {
		res[u.ID] = u
	}

	s.cacheMutex.Lock()
	s.userIdentitiesCache = res
	s.cacheMutex.Unlock()
	return res
}

// GoRead makes slack listening for messages in a different goroutine.
// Those messages will be pushed to the InboxCh of the Pump.
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
					s.InboxCh <- &SlackMessage{ev, s.GetIdentities()}
				}
			case <-s.ReadStopper:
				return
			}
		}
	}()
}

// GoWrite takes care of relaying messages arriving at the outbox
func (s *Slack) GoWrite() {
	go func() {
		for {
			select {
			case msg := <-s.OutboxCh:
				msgOptions, err := msg.ToSlack()
				if err != nil {
					log.Errorln("Slack error converting message to client representation: ", err)
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

// SlackMessage wraps a message event from slack and implements the Message
// Interface
type SlackMessage struct {
	*slack.MessageEvent
	users UserMap
}

// ToSlack is a no-op that returns an error, as we don't want to re-send
// messages from slack to slack at the moment.
func (sm SlackMessage) ToSlack() ([]slack.MsgOption, error) {
	return nil, fmt.Errorf("Messages received in slack are not sent to slack")
}

// ToTelegram converts a received slack message into a proper representation in
// telegram
func (sm SlackMessage) ToTelegram(telegramChatID int64) (telegram.MessageConfig, error) {
	var text string

	if user, ok := sm.users[sm.User]; ok {
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
func (sm SlackMessage) String() string {
	if user, ok := sm.users[sm.User]; ok {
		return fmt.Sprintf("%s: %s", user.Name, sm.Text)

	}
	return fmt.Sprintf("Stranger: %s", sm.Text)
}

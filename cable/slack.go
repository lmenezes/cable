package cable

import (
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

/* Section: Slack type implementing ReadPump() and WritePump() */

// Slack adapts the Telegram Api creating a Pump of messages
type Slack struct {
	// Pump is the pair of Inbox and Outbox channel to receive
	// messages from and write messages to Slack
	*Pump
	// client is the slack api client
	client SlackAPI
	// relayedChannelID is the channel messages will be read from and relayed to
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

// ReadPump makes slack listening for messages in a different goroutine.
// Those messages will be pushed to the Inbox of the Pump.
func (s *Slack) ReadPump() {
	go func() {
		for msg := range s.client.IncomingEvents() {
			switch ev := msg.Data.(type) {
			case *slack.MessageEvent:
				if ev.Channel != s.relayedChannelID || ev.BotID == s.botUserID {
					continue
				}
				s.Inbox <- &SlackMessage{ev, s.GetIdentities()}
			}
		}
	}()
}

// WritePump takes care of relaying messages arriving at the outbox
func (s *Slack) WritePump() {
	go func() {
		for {
			m := <-s.Outbox
			msgOptions, err := m.ToSlack()
			if err != nil {
				log.Errorln("Slack error converting message to client representation: ", err)
			}
			_, _, err = s.client.PostMessage(s.relayedChannelID, msgOptions...)
			if err != nil {
				log.Errorln("Slack error writing message: ", err)
			}
		}
	}()
}

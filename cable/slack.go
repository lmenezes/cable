package cable

import (
	api "github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

// UserMap is a collection of slack Users indexed by their ID, which is a string
type UserMap map[string]api.User

// Slack adapts the Telegram Api creating a Pump of messages
type Slack struct {
	*Pump
	*api.Client
	relayedChannel string
	botUserID      string
}

// NewSlack returns the address of a new value of Slack
func NewSlack(token string, relayedChannel string, botUserID string) *Slack {
	slack := &Slack{
		Pump:           NewPump(),
		Client:         api.New(token),
		relayedChannel: relayedChannel,
		botUserID:      botUserID,
	}

	return slack
}

var userIdentitiesCache UserMap
var cacheMutex sync.Mutex

// GetIdentities returns the list of user information and caches it locally for
// a minute
func (s *Slack) GetIdentities() UserMap {
	cacheMutex.Lock()
	if userIdentitiesCache != nil {
		defer cacheMutex.Unlock()
		return userIdentitiesCache
	}
	cacheMutex.Unlock()

	// clear the cache every 1 minute
	defer func() {
		go func() {
			<-time.NewTimer(1 * time.Minute).C
			log.Debugln("Clearing users cache...")
			cacheMutex.Lock()
			userIdentitiesCache = nil
			cacheMutex.Unlock()
		}()
	}()

	log.Debugln("Setting users cache...")
	users, err := s.Client.GetUsers()
	if err != nil {
		log.Errorln("Cannot get user identities", err)
	}
	res := make(UserMap)
	for _, u := range users {
		res[u.ID] = u
	}

	cacheMutex.Lock()
	userIdentitiesCache = res
	cacheMutex.Unlock()
	return res
}

// ReadPump makes slack listening for messages in a different goroutine.
// Those messages will be pushed to the Inbox of the Pump.
func (s *Slack) ReadPump() {
	rtm := s.NewRTM()
	go rtm.ManageConnection()

	go func() {
		for msg := range rtm.IncomingEvents {
			switch ev := msg.Data.(type) {
			case *api.MessageEvent:
				if ev.Channel != s.relayedChannel || ev.BotID == s.botUserID {
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
				log.Errorln("Slack error converting message to slack representation: ", err)
			}
			_, _, err = s.Client.PostMessage(s.relayedChannel, msgOptions...)
			if err != nil {
				log.Errorln("Slack error writing message: ", err)
			}
		}
	}()
}

package cable

import (
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
	log "github.com/sirupsen/logrus"
)

const (
	// readTimeoutSecs is the number of seconds before timing out the
	// read pump. Timing out implies resetting the connection with the
	// server, which can help in case the connection hung.
	readTimeoutSecs = 60
)

/* Section: Telegram API interface */

// TelegramAPI lets us replace the a telegram-slack-telegram.BotAPI
// with something that behaves like it. This is useful for tests
type TelegramAPI interface {
	GetUpdatesChan(config telegram.UpdateConfig) (telegram.UpdatesChannel, error)
	Send(c telegram.Chattable) (telegram.Message, error)
}

/* Section: Telegram type implementing ReadPump and WritePump */

// Telegram adapts the Telegram Api creating a Pump of messages
type Telegram struct {
	// Pump is the pair of Inbox and Outbox channel to receive
	// messages from and write messages to Telegram
	*Pump
	// client is the telegram API client
	client TelegramAPI
	// relayedChannelID is the channel messages will be read from and relayed to
	relayedChannelID int64
	// botUserID is the id of the slack app installed in the organization, which is
	// used to stop relaying messages posted in telegram itself
	botUserID int
}

// NewTelegram returns the address of a new value of Telegram
func NewTelegram(token string, relayedChannel int64, BotUserID int, debug bool) *Telegram {
	bot, err := telegram.NewBotAPI(token)
	if err != nil {
		log.Fatalln(err)
	}
	bot.Debug = debug
	return &Telegram{
		Pump:             NewPump(),
		client:           bot,
		relayedChannelID: relayedChannel,
		botUserID:        BotUserID,
	}
}

// ReadPump makes telegram listen for messages in a different goroutine.
// Those messages will be pushed to the Inbox of the Pump.
func (t *Telegram) ReadPump() {
	u := telegram.NewUpdate(0)
	u.Timeout = readTimeoutSecs

	updates, err := t.client.GetUpdatesChan(u)
	if err != nil {
		log.Fatalln(err)
	}

	go func() {
		for ev := range updates {
			if ev.Message == nil {
				continue
			}
			msg := ev.Message
			if msg.Chat == nil || msg.Chat.ID != t.relayedChannelID || msg.From.ID == t.botUserID {
				continue
			}
			t.Inbox <- &TelegramMessage{ev}
		}
	}()
}

// WritePump takes care of relaying messages arriving at the outbox
func (t *Telegram) WritePump() {
	go func() {
		for m := range t.Outbox {
			msg, err := m.ToTelegram(t.relayedChannelID)
			if err != nil {
				log.Errorln("Telegram error converting message to telegram representation: ", err)
			}
			_, err = t.client.Send(msg)
			if err != nil {
				log.Errorln("Telegram error writing message: ", err)
			}
		}
	}()
}

package cable

import (
	api "github.com/go-telegram-bot-api/telegram-bot-api"
	log "github.com/sirupsen/logrus"
)

// Telegram adapts the Telegram Api creating a pump of messages
type Telegram struct {
	*Pump
	*api.BotAPI
	relayedChannel int64
	BotUserID      int
}

// NewTelegram returns the address of a new value of Telegram
func NewTelegram(token string, relayedChannel int64, BotUserID int) *Telegram {
	bot, err := api.NewBotAPI(token)
	if err != nil {
		log.Fatalln(err)
	}
	return &Telegram{
		Pump:           newPump(),
		BotAPI:         bot,
		relayedChannel: relayedChannel,
		BotUserID:      BotUserID,
	}
}

// ReadPump makes slack listening for messages in a different goroutine.
// Those messages will be pushed to the Inbox of the Pump.
func (t *Telegram) ReadPump() {
	t.Debug = true

	u := api.NewUpdate(0)
	u.Timeout = 60

	updates, err := t.GetUpdatesChan(u)
	if err != nil {
		log.Fatalln(err)
	}

	go func() {
		for ev := range updates {
			if ev.Message == nil {
				continue
			}
			msg := ev.Message
			if msg.Chat == nil || /* msg.Chat.ID != t.relayedChannel || */ msg.From.ID == t.BotUserID {
				continue
			}
			t.Inbox <- &TelegramMessage{&ev}
		}
	}()
}

// WritePump takes care of relaying messages arriving at the outbox
func (t *Telegram) WritePump() {
	go func() {
		for {
			m := <-t.Outbox
			msg, err := m.ToTelegram(t.relayedChannel)
			if err != nil {
				log.Errorln("Error converting message to telegram representation: ", err)
			}
			_, err = t.Send(msg)
			if err != nil {
				log.Errorln("Error writing message: ", err)
			}
		}
	}()
}

package cable

import (
	api "github.com/go-telegram-bot-api/telegram-bot-api"
	log "github.com/sirupsen/logrus"
)

// TelegramBotApi lets us replace the a telegram-bot-api.BotAPI
// with something that behaves like it. This is useful for tests
type TelegramBotApi interface {
	GetUpdatesChan(config api.UpdateConfig) (api.UpdatesChannel, error)
	Send(c api.Chattable) (api.Message, error)
}

// Telegram adapts the Telegram Api creating a Pump of messages
type Telegram struct {
	*Pump
	bot            TelegramBotApi
	relayedChannel int64
	botUserID      int
}

// NewTelegram returns the address of a new value of Telegram
func NewTelegram(token string, relayedChannel int64, BotUserID int, debug bool) *Telegram {
	bot, err := api.NewBotAPI(token)
	if err != nil {
		log.Fatalln(err)
	}
	bot.Debug = debug
	return &Telegram{
		Pump:           NewPump(),
		bot:            bot,
		relayedChannel: relayedChannel,
		botUserID:      BotUserID,
	}
}

// ReadPump makes slack listening for messages in a different goroutine.
// Those messages will be pushed to the Inbox of the Pump.
func (t *Telegram) ReadPump() {
	u := api.NewUpdate(0)
	u.Timeout = 60

	updates, err := t.bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatalln(err)
	}

	go func() {
		for ev := range updates {
			if ev.Message == nil {
				continue
			}
			msg := ev.Message
			if msg.Chat == nil || msg.Chat.ID != t.relayedChannel || msg.From.ID == t.botUserID {
				continue
			}
			t.Inbox <- &TelegramMessage{ev}
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
				log.Errorln("Telegram error converting message to telegram representation: ", err)
			}
			_, err = t.bot.Send(msg)
			if err != nil {
				log.Errorln("Telegram error writing message: ", err)
			}
		}
	}()
}

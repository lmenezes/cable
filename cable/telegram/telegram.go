package telegram

import (
	"fmt"
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/miguelff/cable/cable"
	"github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
	"strings"
)

const (
	// readTimeoutSecs is the number of seconds before timing out the
	// read pump. Timing out implies resetting the connection with the
	// server, which can help in case the connection died.
	readTimeoutSecs = 60
)

/* Section: Telegram API interface */

// API lets us replace the a telegram-slack-telegram.BotAPI
// with something that behaves like it. This is useful for tests
type API interface {
	GetUpdatesChan(config telegram.UpdateConfig) (telegram.UpdatesChannel, error)
	Send(c telegram.Chattable) (telegram.Message, error)
}

/* Section: Telegram type implementing GoRead and GoWrite */

// Telegram adapts the Telegram API creating a Pump of messages
type Telegram struct {
	// Pump is the pair of InboxCh and OutboxCh channels to receive
	// messages from and write messages to Telegram
	*cable.Pump
	// client is the telegram API client
	client API
	// relayedChatID is the ID of the group chat which messages will be read
	// from and relayed to
	relayedChatID int64
	// botUserID is the id of the telegram app installed, which is used to
	// discard messages looped back by the own bot
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
		Pump:          cable.NewPump(),
		client:        bot,
		relayedChatID: relayedChannel,
		botUserID:     BotUserID,
	}
}

// GoRead makes telegram listen for messages in a different goroutine.
// Those messages will be pushed to the InboxCh of the Pump.
//
// The goroutine can be stopped by feeding ReadStopper synchronization channel
// which can be done by calling StopRead() - a method coming from Pump and
// which is accessed directly through the Telegram value.
func (t *Telegram) GoRead() {
	u := telegram.NewUpdate(0)
	u.Timeout = readTimeoutSecs

	updates, err := t.client.GetUpdatesChan(u)
	if err != nil {
		log.Fatalln(err)
	}

	go func() {
		for {
			select {
			case ev := <-updates:
				if ev.Message == nil {
					continue
				}
				msg := ev.Message
				if msg.Chat == nil || msg.Chat.ID != t.relayedChatID || msg.From.ID == t.botUserID {
					continue
				}
				t.Inbox() <- &Message{ev}
			case <-t.ReadStopper:
				return
			}
		}
	}()
}

// GoWrite spawns a goroutine that takes care of delivering to telegram the
// messages arriving at the OutboxCh of the Pump.
//
// The goroutine can be stopped by feeding WriteStopper synchronization channel
// which can be done by calling StopWrite() - a method coming from Pump and
// which is accessed directly through the Telegram value.
func (t *Telegram) GoWrite() {
	go func() {
		for {
			select {
			case m := <-t.Outbox():
				msg, err := m.ToTelegram(t.relayedChatID)
				if err != nil {
					log.Errorln("Telegram error converting message to telegram representation: ", err)
				}
				_, err = t.client.Send(msg)
				if err != nil {
					log.Errorln("Telegram error writing message: ", err)
				}
			case <-t.WriteStopper:
				return
			}
		}
	}()
}

/* Telegram message */

// Message wraps a telegram update and implements the Message Interface
type Message struct {
	telegram.Update
}

// ToSlack converts a received telegram message into a proper representation in
// slack
func (tm Message) ToSlack() ([]slack.MsgOption, error) {
	var authorName []string

	if firstName := tm.Update.Message.From.FirstName; firstName != "" {
		authorName = append(authorName, firstName)
	}

	if lastName := tm.Update.Message.From.LastName; lastName != "" {
		authorName = append(authorName, lastName)
	}

	if userName := tm.Update.Message.From.UserName; userName != "" {
		if len(authorName) > 0 {
			authorName = append(authorName, fmt.Sprintf("(%s)", userName))
		} else {
			authorName = append(authorName, userName)
		}
	}

	attachment := slack.Attachment{
		Fallback:   tm.Message.Text,
		AuthorName: strings.Join(authorName, " "),
		Text:       tm.Message.Text,
	}

	return []slack.MsgOption{slack.MsgOptionAttachments(attachment)}, nil
}

// ToTelegram is a no-op that returns an error as we dont want to re-send
// messages from telegram to telegram at the moment
func (tm Message) ToTelegram(telegramChatID int64) (telegram.MessageConfig, error) {
	return telegram.MessageConfig{}, fmt.Errorf("Messages received in telegram are not sent back to telegram")
}

// String returns a human readable representation of a telegram message for
// debugging purposes
func (tm Message) String() string {
	return fmt.Sprintf("%s: %s", tm.Update.Message.From.UserName, tm.Message.Text)
}

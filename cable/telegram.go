package cable

import (
	"fmt"
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
	"strings"
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

/* Section: Telegram type implementing GoRead and GoWrite */

// Telegram adapts the Telegram Api creating a Pump of messages
type Telegram struct {
	// Pump is the pair of InboxCh and OutboxCh channel to receive
	// messages from and write messages to Telegram
	*Pump
	// client is the telegram API client
	client TelegramAPI
	// relayedChatID is the ID of the chat messages will be read from and
	// relayed to
	relayedChatID int64
	// botUserID is the id of the slack app installed in the organization,
	// which is used to stop relaying messages posted in telegram itself
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
		Pump:          NewPump(),
		client:        bot,
		relayedChatID: relayedChannel,
		botUserID:     BotUserID,
	}
}

// GoRead makes telegram listen for messages in a different goroutine.
// Those messages will be pushed to the InboxCh of the Pump.
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
				t.InboxCh <- &TelegramMessage{ev}
			case <-t.ReadStopper:
				return
			}
		}
	}()
}

// GoWrite takes care of relaying messages arriving at the outbox
func (t *Telegram) GoWrite() {
	go func() {
		for {
			select {
			case m := <-t.OutboxCh:
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

// TelegramMessage wraps a telegram update and implements the Message Interface
type TelegramMessage struct {
	telegram.Update
}

// ToSlack converts a received telegram message into a proper representation in
// slack
func (tm TelegramMessage) ToSlack() ([]slack.MsgOption, error) {
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
func (tm TelegramMessage) ToTelegram(telegramChatID int64) (telegram.MessageConfig, error) {
	return telegram.MessageConfig{}, fmt.Errorf("Messages received in telegram are not sent back to telegram")
}

// String returns a human readable representation of a telegram message for
// debugging purposes
func (tm TelegramMessage) String() string {
	return fmt.Sprintf("%s: %s", tm.Update.Message.From.UserName, tm.Message.Text)
}

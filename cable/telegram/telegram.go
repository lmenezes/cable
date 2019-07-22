package telegram

import (
	"fmt"
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/kyokomi/emoji"
	"github.com/miguelff/cable/cable"
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
				update, err := t.ToInboxUpdate(ev)
				if err != nil {
					log.Debugf("Update from inbox discarded: %s", err)
				} else {
					t.Inbox() <- update
				}
			case <-t.ReadStopper:
				return
			}
		}
	}()
}

// ToInboxUpdate converts any event received in the read pump to a cable update
// that can be fed into the inbox
func (t *Telegram) ToInboxUpdate(update interface{}) (cable.Update, error) {
	ev := update.(telegram.Update)
	if ev.Message != nil {
		return t.toInboxMessage(ev.Message)
	}
	return nil, fmt.Errorf("ignoring unknown update type %s", update)
}

func (t *Telegram) toInboxMessage(msg *telegram.Message) (cable.Update, error) {
	if msg.Chat == nil || msg.Chat.ID != t.relayedChatID || msg.From.ID == t.botUserID {
		return nil, fmt.Errorf("ignoring message: %s", msg.Text)
	}
	var authorName []string

	if firstName := msg.From.FirstName; firstName != "" {
		authorName = append(authorName, firstName)
	}

	if lastName := msg.From.LastName; lastName != "" {
		authorName = append(authorName, lastName)
	}

	return cable.Message{
		Author: &cable.Author{
			Name:  strings.Join(authorName, " "),
			Alias: msg.From.UserName,
		},
		Contents: &cable.Contents{Raw: msg.Text},
	}, nil
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
			case ou := <-t.Outbox():
				update, err := t.FromOutboxUpdate(ou)
				if err != nil {
					log.Debugf("Update from inbox discarded: %s", err)
				}
				err = t.Send(update)
				if err != nil {
					log.Errorln("Error sending message: ", err)
				}
			case <-t.WriteStopper:
				return
			}
		}
	}()
}

// Send sends a telegram update
func (t *Telegram) Send(update interface{}) error {
	_, err := t.client.Send(update.(telegram.Chattable))
	return err
}

// FromOutboxUpdate converts the given Update into a telegram.Chattable message, which
// this pumper know how to send over the write
func (t *Telegram) FromOutboxUpdate(update cable.Update) (interface{}, error) {
	switch ir := update.(type) {
	case cable.Message:
		return t.fromOutboxMessage(ir), nil
	default:
		return nil, fmt.Errorf("cannot convert update to telegram: %s ", update)
	}
}

func (t *Telegram) fromOutboxMessage(m cable.Message) telegram.Chattable {
	var contents string
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
		contents = fmt.Sprintf("*%s:* %s", strings.Join(authorTokens, " "), m.Contents)
	} else {
		contents = m.Contents.String()
	}

	return telegram.MessageConfig{
		BaseChat: telegram.BaseChat{
			ChatID:           t.relayedChatID,
			ReplyToMessageID: 0,
		},
		Text:                  emoji.Sprint(contents),
		DisableWebPagePreview: false,
		ParseMode:             telegram.ModeMarkdown,
	}
}

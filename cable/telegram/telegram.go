package telegram

import (
	"fmt"
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/kyokomi/emoji"
	"github.com/miguelff/cable/cable"
	log "github.com/sirupsen/logrus"
	"strings"
	"sync"
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

/*

	Section: Telegram type embedding cable.Pump and thus implementing part of the
	Pumper interface implicitly.

*/

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
	// the channel where updates will arrive to
	updatesCh telegram.UpdatesChannel
	// updatesChMutex mutex to synchronize access to updatesCh
	updatesChMutex sync.Mutex
}

// NewTelegram returns the address of a new value of Telegram
func NewTelegram(token string, relayedChannel int64, BotUserID int, debug bool) *Telegram {
	bot, err := telegram.NewBotAPI(token)
	if err != nil {
		log.Fatalln(err)
	}
	bot.Debug = debug

	t := &Telegram{
		client:        bot,
		relayedChatID: relayedChannel,
		botUserID:     BotUserID,
	}
	cable.NewPump(t)
	return t
}

/* Pumper interface */

// NextEvent blocks until it reads something from the telegram api
func (t *Telegram) NextEvent() cable.Update {
	t.updatesChMutex.Lock()
	if t.updatesCh == nil {
		// lazily initialize the channel
		u := telegram.NewUpdate(0)
		u.Timeout = readTimeoutSecs
		ch, err := t.client.GetUpdatesChan(u)
		if err != nil {
			log.Panic(err)
		}
		t.updatesCh = ch
	}
	t.updatesChMutex.Unlock()
	return <-t.updatesCh
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

// Send sends a telegram update
func (t *Telegram) Send(update interface{}) error {
	_, err := t.client.Send(update.(telegram.Chattable))
	return err
}

/* Private methods */

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

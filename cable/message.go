package cable

import (
	"fmt"
	t "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/kyokomi/emoji"
	s "github.com/nlopes/slack"
	"strings"
)

// Message a standard representation of a message
type Message interface {
	ToSlack() ([]s.MsgOption, error)
	ToTelegram(telegramChatID int64) (t.MessageConfig, error)
	String() string
}

// SlackMessage wraps a message event from slack and implements the Message
// Interface
type SlackMessage struct {
	*s.MessageEvent
	users UserMap
}

// ToSlack is a no-op that returns an error, as we don't want to re-send
// messages from slack to slack at the moment.
func (sm *SlackMessage) ToSlack() ([]s.MsgOption, error) {
	return nil, fmt.Errorf("Messages received in slack are not sent to slack")
}

// ToTelegram converts a received slack message into a proper representation in
// telegram
func (sm *SlackMessage) ToTelegram(telegramChatID int64) (t.MessageConfig, error) {
	var text string

	if user, ok := sm.users[sm.User]; ok {
		text = fmt.Sprintf("*%s (%s):* %s", user.RealName, user.Name, sm.Text)
	} else {
		text = fmt.Sprintf("*Stranger:* %s", sm.Text)
	}

	return t.MessageConfig{
		BaseChat: t.BaseChat{
			ChatID:           telegramChatID,
			ReplyToMessageID: 0,
		},
		Text:                  emoji.Sprint(text),
		DisableWebPagePreview: false,
		ParseMode:             t.ModeMarkdown,
	}, nil
}

// String returns a human readable representation of a slack message for
// debugging purposes
func (sm *SlackMessage) String() string {
	if user, ok := sm.users[sm.User]; ok {
		return fmt.Sprintf("%s: %s", user.Name, sm.Text)

	}
	return fmt.Sprintf("Stranger: %s", sm.Text)
}

// TelegramMessage wraps a telegram update and implements the Message Interface
type TelegramMessage struct {
	*t.Update
}

// ToSlack converts a received telegram message into a proper representation in
// slack
func (tm *TelegramMessage) ToSlack() ([]s.MsgOption, error) {
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

	attachment := s.Attachment{
		Fallback:   tm.Message.Text,
		AuthorName: strings.Join(authorName, " "),
		Text:       tm.Message.Text,
	}

	return []s.MsgOption{s.MsgOptionAttachments(attachment)}, nil
}

// ToTelegram is a no-op that returns an error, as we dont want to re-send
// messages from telegram to telegram at the moment
func (tm *TelegramMessage) ToTelegram(telegramChatID int64) (t.MessageConfig, error) {
	return t.MessageConfig{}, fmt.Errorf("Messages received in telegram are not sent back to telegram")
}

// String returns a human readable representation of a telegram message for
// debugging purposes
func (tm *TelegramMessage) String() string {
	return fmt.Sprintf("%s: %s", tm.Update.Message.From.UserName, tm.Message.Text)
}

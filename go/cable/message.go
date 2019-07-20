package cable

import (
	"fmt"
	t "github.com/go-telegram-bot-api/telegram-bot-api"
	s "github.com/nlopes/slack"
	"strings"
)

// Message a standard representation of a message
type Message interface {
	ToSlack() ([]s.MsgOption, error)
	ToTelegram() (string, error)
}

// SlackMessage wraps a message event from slack and implements the Message
// Interface
type SlackMessage struct {
	*s.MessageEvent
	UserList map[string]s.User
}

// ToSlack is a no-op that returns an error, as we don't want to re-send
// messages from slack to slack at the moment.
func (sm *SlackMessage) ToSlack() ([]s.MsgOption, error) {
	return nil, fmt.Errorf("Messages received in slack are not sent to slack")
}

// ToTelegram converts a received slack message into a proper representation in
// telegram
func (sm *SlackMessage) ToTelegram() (string, error) {
	userID := sm.User

	if user, ok := sm.UserList[userID]; ok {
		return fmt.Sprintf("%s (%s): %s", user.RealName, user.Name, sm.Text), nil

	}
	return fmt.Sprintf("Stranger: %s", sm.Text), nil

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
func (tm *TelegramMessage) ToTelegram() (string, error) {
	return "", fmt.Errorf("Messages received in telegram are not sent back to telegram")
}

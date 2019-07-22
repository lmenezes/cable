package cable

import (
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/nlopes/slack"
)

// Message is the interface any message interchanged by a Pumper has to
// implement
type Message interface {
	ToSlack() ([]slack.MsgOption, error)
	ToTelegram(telegramChatID int64) (telegram.MessageConfig, error)
	String() string
}

package cable

import (
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/nlopes/slack"
	. "github.com/stretchr/testify/assert"
	"testing"
)

func TestSlackMessage_String_KnownUser(t *testing.T) {
	user := slack.User{ID: slackUserID, RealName: "Will Smith", Name: "freshprince"}
	msg := createSlackMessage("Sup Jay!", slackUserID, user)
	Equal(t, "freshprince: Sup Jay!", msg.String())
}

func TestSlackMessage_String_Stranger(t *testing.T) {
	user := slack.User{ID: slackUserID, RealName: "Will Smith", Name: "freshprince"}
	msg := createSlackMessage("Sup Jay!", unknownSlackUSerID, user)
	Equal(t, "Stranger: Sup Jay!", msg.String())
}

func TestSlackMessage_ToSlack(t *testing.T) {
	user := slack.User{ID: slackUserID, RealName: "Will Smith", Name: "freshprince"}
	msg := createSlackMessage("Sup Jay!", slackUserID, user)
	_, e := msg.ToSlack()
	Error(t, e)
}

func TestSlackMessage_ToTelegram_KnownUser(t *testing.T) {
	user := slack.User{ID: slackUserID, RealName: "Will Smith", Name: "freshprince"}
	msg := createSlackMessage("Sup Jay! :boom:", slackUserID, user)
	telegramChatID := int64(123)

	expected := telegram.MessageConfig{
		BaseChat: telegram.BaseChat{
			ChatID:           telegramChatID,
			ReplyToMessageID: 0,
		},
		Text:                  "*Will Smith (freshprince):* Sup Jay! 💥 ",
		DisableWebPagePreview: false,
		ParseMode:             "Markdown",
	}

	actual, _ := msg.ToTelegram(telegramChatID)
	Equal(t, expected, actual)
}

func TestSlackMessage_ToTelegram_Stranger(t *testing.T) {
	user := slack.User{ID: slackUserID, RealName: "Will Smith", Name: "freshprince"}
	msg := createSlackMessage("Sup Jay! :boom:", "STRGRID", user)
	telegramChatID := int64(123)

	expected := telegram.MessageConfig{
		BaseChat: telegram.BaseChat{
			ChatID:           telegramChatID,
			ReplyToMessageID: 0,
		},
		Text:                  "*Stranger:* Sup Jay! 💥 ",
		DisableWebPagePreview: false,
		ParseMode:             "Markdown",
	}

	actual, _ := msg.ToTelegram(telegramChatID)
	Equal(t, expected, actual)
}

func TestTelegramMessage_String(t *testing.T) {
	msg := createTelegramMessage("Sup will! pss", "Jeffrey", "Townes", "Jazz")
	Equal(t, "Jazz: Sup will! pss", msg.String())
}

func TestTelegramMessage_ToSlack(t *testing.T) {
	msg := createTelegramMessage("Sup will! :punch: :thumbs_up:", "Jeffrey", "Townes", "Jazz")
	slackMessages, _ := msg.ToSlack()

	jsonMessage := asSlackJSONMessage(slackMessages[0])
	actual := jsonMessage

	expected := slackJSONMessage{
		Fallback:   "Sup will! :punch: :thumbs_up:",
		AuthorName: "Jeffrey Townes (Jazz)",
		Text:       "Sup will! :punch: :thumbs_up:",
	}
	Equal(t, expected, actual)
}

func TestTelegramMessage_ToTelegram(t *testing.T) {
	msg := createTelegramMessage("Sup will! :punch: :thumbs_up:", "Jeffrey", "Townes", "Jazz")
	telegramChatID := int64(123)
	_, e := msg.ToTelegram(telegramChatID)
	Error(t, e)
}

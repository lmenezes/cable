package cable

import (
	"encoding/json"
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/nlopes/slack"
	. "github.com/stretchr/testify/assert"
	"testing"
)

// createSlackMessage is a helper factory of slack messages for the tests below
func createSlackMessage(text string, authorID string, worksSpaceUsers ...slack.User) SlackMessage {
	users := make(UserMap)
	for _, u := range worksSpaceUsers {
		users[u.ID] = u
	}

	return SlackMessage{
		MessageEvent: slack.MessageEvent{Msg: slack.Msg{User: authorID, Text: text}},
		users:        users,
	}
}

// createSlackUser is a helper factory of slack users for the tests below
func createSlackUser(ID string, realName string, nickname string) slack.User {
	return slack.User{ID: ID, RealName: realName, Name: nickname}

}

// createTelegramMessage is a helper factory of telegram messages for the tests below
func createTelegramMessage(text string, authorFirstName string, authorLastName string, authorUserName string) TelegramMessage {
	return TelegramMessage{
		telegram.Update{
			Message: &telegram.Message{
				From: &telegram.User{
					FirstName: authorFirstName,
					LastName:  authorLastName,
					UserName:  authorUserName,
				},
				Text: text,
			},
		},
	}
}

func TestSlackMessage_String_KnownUser(t *testing.T) {
	msg := createSlackMessage("Sup Jay!", "0001", createSlackUser("0001", "Will Smith", "freshprince"))
	Equal(t, "freshprince: Sup Jay!", msg.String())
}

func TestSlackMessage_String_Stranger(t *testing.T) {
	msg := createSlackMessage("Sup Jay!", "STRGRID", createSlackUser("0001", "Will Smith", "freshprince"))
	Equal(t, "Stranger: Sup Jay!", msg.String())
}

func TestSlackMessage_ToSlack(t *testing.T) {
	msg := createSlackMessage("Sup Jay!", "0001", createSlackUser("0001", "Will Smith", "freshprince"))
	_, e := msg.ToSlack()
	Error(t, e)
}

func TestSlackMessage_ToTelegram_KnownUser(t *testing.T) {
	msg := createSlackMessage("Sup Jay! :boom:", "0001", createSlackUser("0001", "Will Smith", "freshprince"))
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
	msg := createSlackMessage("Sup Jay! :boom:", "STRGRID", createSlackUser("0001", "Will Smith", "freshprince"))
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
	type slackJSONMessage struct {
		Fallback   string `json:"fallback"`
		AuthorName string `json:"author_name"`
		Text       string `json:"text"`
	}

	msg := createTelegramMessage("Sup will! :punch: :thumbs_up:", "Jeffrey", "Townes", "Jazz")
	slackMessage, _ := msg.ToSlack()
	// What slack can send is not really a struct but a closure that when called given a context
	// returns the HTML payload to send. slack.UnsafeApplyMsgOptions let us debug this.
	_, configuration, _ := slack.UnsafeApplyMsgOptions("SAMPLE_TOKEN", "SAMPLE_CHANNEL", slackMessage...)
	serializedAttachments := []byte(configuration["attachments"][0])
	jsonMessages := []slackJSONMessage{}
	json.Unmarshal(serializedAttachments, &jsonMessages)
	actual := jsonMessages[0]

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

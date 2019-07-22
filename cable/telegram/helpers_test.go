package telegram

import (
	"encoding/json"
	telegramAPI "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/miguelff/cable/cable/slack"
	slackAPI "github.com/nlopes/slack"
)

/* Constants used in tests */

const (
	telegramBotID = iota
	telegramUserID
	telegramChatID
	unknownTelegramChatID
)

/* fake Telegram API */

type fakeTelegramAPI struct {
	updatesChannel telegramAPI.UpdatesChannel
	sent           []telegramAPI.Chattable
}

func (api *fakeTelegramAPI) GetUpdatesChan(config telegramAPI.UpdateConfig) (telegramAPI.UpdatesChannel, error) {
	return api.updatesChannel, nil
}

func (api *fakeTelegramAPI) Send(c telegramAPI.Chattable) (telegramAPI.Message, error) {
	api.sent = append(api.sent, c)
	return telegramAPI.Message{}, nil
}

/* factories */

// createUpdate creates a message update as if it was written in the
// relayChannel, by a user with the given UserID, and with the given text
func createTelegramBotUpdate(relayedChannelID int64, text string) telegramAPI.Update {
	return telegramAPI.Update{
		Message: &telegramAPI.Message{
			Text: text,
			Chat: &telegramAPI.Chat{
				ID: relayedChannelID,
			},
			From: &telegramAPI.User{
				ID:       telegramBotID,
				UserName: "CableBot",
			},
		},
	}
}

// createUpdate creates a message update as if it was written in the
// relayChannel, by a user with the given UserID, and with the given text
func createTelegramUserUpdate(relayedChannel int64, text string) telegramAPI.Update {
	return telegramAPI.Update{
		Message: &telegramAPI.Message{
			Text: text,
			Chat: &telegramAPI.Chat{
				ID: relayedChannel,
			},
			From: &telegramAPI.User{
				ID:       telegramUserID,
				UserName: "freshprince",
			},
		},
	}
}

// createTelegramMessage is a factory of cable.Message for the tests below
func createTelegramMessage(text string, authorFirstName string, authorLastName string, authorUserName string) Message {
	return Message{
		Update: telegramAPI.Update{
			Message: &telegramAPI.Message{
				From: &telegramAPI.User{
					FirstName: authorFirstName,
					LastName:  authorLastName,
					UserName:  authorUserName,
				},
				Text: text,
			},
		},
	}
}

// createSlackMessage is a factory of cable.Message for the tests below
func createSlackMessage(text string, authorID string, worksSpaceUsers ...slackAPI.User) slack.Message {
	users := make(slack.UserMap)
	for _, u := range worksSpaceUsers {
		users[u.ID] = u
	}

	return slack.Message{
		MessageEvent: &slackAPI.MessageEvent{Msg: slackAPI.Msg{User: authorID, Text: text}},
		Users:        users,
	}
}

// slackJSONMessage is a struct used to decode slack.MsgOption
// values for easier management in tests
type slackJSONMessage struct {
	Fallback   string `json:"fallback"`
	AuthorName string `json:"author_name"`
	Text       string `json:"text"`
}

// asSlackJSONMessage converts slack.MsgOption as slackJSONMessage that are
// simpler to use in tests.
//
// This method receives a slice slack.MsgOption, which is what Slack can send
// through its API. slack.MsgOption is not really a struct but a closure that
// when called given a context returns the HTML payload to send.
//
// slack.UnsafeApplyMsgOptions lets obtain the JSON representation of the data
// being send, and we decode it into []slackJSONMessage to use it more easily in
// tests
func asSlackJSONMessage(slackMessages slackAPI.MsgOption) slackJSONMessage {
	jsonMessages := []slackJSONMessage{}
	_, configuration, _ := slackAPI.UnsafeApplyMsgOptions("SAMPLE_TOKEN", "SAMPLE_CHANNEL", slackMessages)
	serializedAttachments := configuration["attachments"][0]
	_ = json.Unmarshal([]byte(serializedAttachments), &jsonMessages)
	return jsonMessages[0]
}

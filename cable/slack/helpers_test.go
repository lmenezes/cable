package slack

import (
	"encoding/json"
	telegramAPI "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/miguelff/cable/cable/telegram"
	slackAPI "github.com/nlopes/slack"
)

/* Constants used in tests */

const (
	slackUserID           = "USER"
	unknownSlackUSerID    = "UNKOWN_USER"
	slackBotID            = "BOT"
	slackChannelID        = "CHANNEL"
	unknownSlackChannelID = "UNKOWN_CHANNEL"
)

/* fake Slack API */

type fakeSlackAPI struct {
	rtmEvents chan slackAPI.RTMEvent
	sent      []slackAPI.MsgOption
	users     UserMap
}

func (api *fakeSlackAPI) IncomingEvents() <-chan slackAPI.RTMEvent {
	return api.rtmEvents
}

func (api *fakeSlackAPI) PostMessage(channelID string, options ...slackAPI.MsgOption) (string, string, error) {
	api.sent = append(api.sent, options...)
	return "", "", nil
}

func (api *fakeSlackAPI) GetUsers() UserMap {
	return api.users
}

/* factories */

// createTelegramMessage is a factory of cable.Message for the tests below
func createTelegramMessage(text string, authorFirstName string, authorLastName string, authorUserName string) telegram.Message {
	return telegram.Message{
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

// createSlackBotUpdate creates a slackAPI.RTM update (what slack reads from the
// API in the read pump) as if it was written by the slack bot itself.
func createSlackBotUpdate(relayedChannelID string, text string) slackAPI.RTMEvent {
	return slackAPI.RTMEvent{
		Data: &slackAPI.MessageEvent{
			Msg: slackAPI.Msg{
				Text:    text,
				Channel: relayedChannelID,
				BotID:   slackBotID,
			},
		},
	}
}

// createSlackUserUpdate creates a slackAPI.RTM update (what slack reads from the
// API in the read pump) as if it was written by a regular user.
func createSlackUserUpdate(relayedChannelID string, text string) slackAPI.RTMEvent {
	return slackAPI.RTMEvent{
		Data: &slackAPI.MessageEvent{
			Msg: slackAPI.Msg{
				User:    slackUserID,
				Text:    text,
				Channel: relayedChannelID,
			},
		},
	}
}

// createSlackMessage is a factory of cable.Message for the tests below
func createSlackMessage(text string, authorID string, worksSpaceUsers ...slackAPI.User) Message {
	users := make(UserMap)
	for _, u := range worksSpaceUsers {
		users[u.ID] = u
	}

	return Message{
		MessageEvent: &slackAPI.MessageEvent{Msg: slackAPI.Msg{User: authorID, Text: text}},
		Users:        users,
	}
}

// createSlackUser is a factory of slack Users
func createSlackUser(userID string, realName string, username string) slackAPI.User {
	return slackAPI.User{ID: userID, RealName: realName, Name: username}
}

// slackJSONMessage is a struct used to decode slackAPI.MsgOption
// values for easier management in tests
type slackJSONMessage struct {
	Fallback   string `json:"fallback"`
	AuthorName string `json:"author_name"`
	Text       string `json:"text"`
}

// asSlackJSONMessage converts slackAPI.MsgOption as slackJSONMessage that are
// simpler to use in tests.
//
// This method receives a slice slackAPI.MsgOption, which is what Slack can send
// through its API. slackAPI.MsgOption is not really a struct but a closure that
// when called given a context returns the HTML payload to send.
//
// slackAPI.UnsafeApplyMsgOptions lets obtain the JSON representation of the data
// being send, and we decode it into []slackJSONMessage to use it more easily in
// tests
func asSlackJSONMessage(slackMessages slackAPI.MsgOption) slackJSONMessage {
	jsonMessages := []slackJSONMessage{}
	_, configuration, _ := slackAPI.UnsafeApplyMsgOptions("SAMPLE_TOKEN", "SAMPLE_CHANNEL", slackMessages)
	serializedAttachments := configuration["attachments"][0]
	_ = json.Unmarshal([]byte(serializedAttachments), &jsonMessages)
	return jsonMessages[0]
}

package cable

import (
	"encoding/json"
	"fmt"
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/nlopes/slack"
)

/* Constants used in tests */

const (
	slackUserID           = "USER"
	unknownSlackUSerID    = "UNKOWN_USER"
	slackBotID            = "BOT"
	slackChannelID        = "CHANNEL"
	unknownSlackChannelID = "UNKOWN_CHANNEL"

	telegramBotID = iota
	telegramUserID
	telegramChatID
	unknownTelegramChatID
)

/* fake pump */

type fakePumper struct {
	*Pump
}

func newFakePumper() *fakePumper {
	return &fakePumper{NewPump()}
}

func (*fakePumper) GoRead() {}

func (*fakePumper) GoWrite() {}

/* fake Message */

type fakeMessage struct {
	text string
}

func (fm fakeMessage) ToSlack() ([]slack.MsgOption, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (fm fakeMessage) ToTelegram(telegramChatID int64) (telegram.MessageConfig, error) {
	return telegram.MessageConfig{}, fmt.Errorf("Not implemented")
}

func (fm fakeMessage) String() string {
	return fm.text
}

/* fake Slack API */

type fakeSlackAPI struct {
	rtmEvents chan slack.RTMEvent
	sent      []slack.MsgOption
	users     []slack.User
}

func (api *fakeSlackAPI) IncomingEvents() <-chan slack.RTMEvent {
	return api.rtmEvents
}

func (api *fakeSlackAPI) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
	for _, msg := range options {
		api.sent = append(api.sent, msg)
	}
	return "", "", nil
}

func (api *fakeSlackAPI) GetUsers() ([]slack.User, error) {
	return api.users, nil
}

/* fake Telegram API */

type fakeTelegramAPI struct {
	updatesChannel telegram.UpdatesChannel
	sent           []telegram.Chattable
}

func (api *fakeTelegramAPI) GetUpdatesChan(config telegram.UpdateConfig) (telegram.UpdatesChannel, error) {
	return api.updatesChannel, nil
}

func (api *fakeTelegramAPI) Send(c telegram.Chattable) (telegram.Message, error) {
	api.sent = append(api.sent, c)
	return telegram.Message{}, nil
}

/* factories */

// createUpdate creates a message update as if it was written in the
// relayChannel, by a user with the given UserID, and with the given text
func createTelegramBotUpdate(relayedChannelID int64, text string) telegram.Update {
	return telegram.Update{
		Message: &telegram.Message{
			Text: text,
			Chat: &telegram.Chat{
				ID: relayedChannelID,
			},
			From: &telegram.User{
				ID:       telegramBotID,
				UserName: "CableBot",
			},
		},
	}
}

// createUpdate creates a message update as if it was written in the
// relayChannel, by a user with the given UserID, and with the given text
func createTelegramUserUpdate(relayedChannel int64, text string) telegram.Update {
	return telegram.Update{
		Message: &telegram.Message{
			Text: text,
			Chat: &telegram.Chat{
				ID: relayedChannel,
			},
			From: &telegram.User{
				ID:       telegramUserID,
				UserName: "freshprince",
			},
		},
	}
}

// createTelegramMessage is a factory of cable.TelegramMessage for the tests below
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

// createSlackBotUpdate creates a slack.RTM update (what slack reads from the
// API in the read pump) as if it was written by the slack bot itself.
func createSlackBotUpdate(relayedChannelID string, text string) slack.RTMEvent {
	return slack.RTMEvent{
		Data: &slack.MessageEvent{
			Msg: slack.Msg{
				Text:    text,
				Channel: relayedChannelID,
				BotID:   slackBotID,
			},
		},
	}
}

// createSlackUserUpdate creates a slack.RTM update (what slack reads from the
// API in the read pump) as if it was written by a regular user.
func createSlackUserUpdate(relayedChannelID string, text string) slack.RTMEvent {
	return slack.RTMEvent{
		Data: &slack.MessageEvent{
			Msg: slack.Msg{
				User:    slackUserID,
				Text:    text,
				Channel: relayedChannelID,
			},
		},
	}
}

// createSlackMessage is a factory of cable.SlackMessage for the tests below
func createSlackMessage(text string, authorID string, worksSpaceUsers ...slack.User) SlackMessage {
	users := make(UserMap)
	for _, u := range worksSpaceUsers {
		users[u.ID] = u
	}

	return SlackMessage{
		MessageEvent: &slack.MessageEvent{Msg: slack.Msg{User: authorID, Text: text}},
		users:        users,
	}
}

// createSlackUser is a factory of slack users
func createSlackUser(userID string, realName string, username string) slack.User {
	return slack.User{ID: userID, RealName: realName, Name: username}
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
// slack.UnsafeApplyMsgOptions let obtain the JSON representation of the data
// being send, and we decode it into []slackJSONMessage to use it more easily in
// tests
func asSlackJSONMessage(slackMessages slack.MsgOption) slackJSONMessage {
	jsonMessages := []slackJSONMessage{}
	_, configuration, _ := slack.UnsafeApplyMsgOptions("SAMPLE_TOKEN", "SAMPLE_CHANNEL", slackMessages)
	serializedAttachments := configuration["attachments"][0]
	_ = json.Unmarshal([]byte(serializedAttachments), &jsonMessages)
	return jsonMessages[0]
}

package cable

import (
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/nlopes/slack"
	. "github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSlack_ReadPump(t *testing.T) {
	updates := []slack.RTMEvent{
		createSlackBotUpdate(slackChannelID, "Hey Hey!"),                           // discarded, because written by the bot itself
		createSlackUserUpdate(slackChannelID, "Sup Jay!"),                          // selected
		createSlackUserUpdate(unknownSlackChannelID, "Uncle Phil, where are you?"), // discarded because written by a user in a chat other than the relayed channel
		createSlackUserUpdate(slackChannelID, "Uncle Phil, you here?"),             // selected
		{}, // discarded: no message
	}

	updatesCh := make(chan slack.RTMEvent, len(updates))
	for _, update := range updates {
		updatesCh <- update
	}

	fakeSlack := &Slack{
		relayedChannelID: slackChannelID,
		botUserID:        slackBotID,
		client: &fakeSlackAPI{
			rtmEvents: updatesCh,
			users:     []slack.User{createSlackUser(slackUserID, "Will Smith", "freshprince")},
		},
		Pump: NewPump(),
	}

	fakeSlack.GoRead()

	// wait for the pump to to process the channel up to 1 second, or timeout
	timeout := time.NewTimer(1 * time.Second)
WAIT:
	for {
		select {
		case <-timeout.C:
			Fail(t, "timeout while processing the Read Pump")
			break WAIT
		default:
			if len(updatesCh) == 0 {
				close(fakeSlack.InboxCh)
				break WAIT
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	fakeSlack.StopRead()

	var inbox []Message
	for message := range fakeSlack.InboxCh {
		inbox = append(inbox, message)
	}

	Equal(t, 2, len(inbox))
	Equal(t, "freshprince: Sup Jay!", inbox[0].String())
	Equal(t, "freshprince: Uncle Phil, you here?", inbox[1].String())
}

func TestSlack_WritePump(t *testing.T) {
	client := &fakeSlackAPI{}

	fakeSlack := &Slack{
		relayedChannelID: slackChannelID,
		botUserID:        slackBotID,
		client:           client,
		Pump:             NewPump(),
	}

	fakeSlack.OutboxCh <- createTelegramMessage("Sup Jay!", "Will", "Smith", "freshprince")
	fakeSlack.OutboxCh <- createTelegramMessage(":clap: Psss!", "Will", "Smith", "freshprince")

	fakeSlack.GoWrite()

	// wait for the pump to to process the channel up to 1 second, or timeout
	timeout := time.NewTimer(1 * time.Second)

WAIT:
	for {
		select {
		case <-timeout.C:
			Fail(t, "timeout while processing the Read Pump")
			break WAIT
		default:
			if len(fakeSlack.OutboxCh) == 0 {
				break WAIT
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	fakeSlack.StopWrite()

	Equal(t, 2, len(client.sent))

	first := asSlackJSONMessage(client.sent[0])
	Equal(t, "Sup Jay!", first.Text)

	second := asSlackJSONMessage(client.sent[1])
	Equal(t, ":clap: Psss!", second.Text)
}

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

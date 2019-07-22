package cable

import (
	"github.com/nlopes/slack"
	. "github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSlack_ReadPump(t *testing.T) {
	updates := []slack.RTMEvent{
		{}, // message not set, discarded
		createSlackBotUpdate(slackChatID, "Hey Hey!"),                          // discarded, update by the bot
		createSlackUserUpdate(slackChatID, "Sup Jay!"),                         // selected, by a user in the relayed channel
		createSlackUserUpdate(unkownSlackChatID, "Uncle Phil, where are you?"), // discarded, by a user in a chat other than the relayed channel
		createSlackUserUpdate(slackChatID, "Uncle Phil, you here?"),            // discarded, by a user in a chat other than the relayed channel
	}

	updatesCh := make(chan slack.RTMEvent, len(updates))
	for _, update := range updates {
		updatesCh <- update
	}

	users := []slack.User{createSlackUser(slackUserID, "Will Smith", "freshprince")}

	fakeSlack := &Slack{
		relayedChannelID: slackChatID,
		botUserID:        slackBotID,
		client:           &fakeSlackAPI{rtmEvents: updatesCh, users: users},
		Pump:             NewPump(),
	}

	fakeSlack.ReadPump()

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
				close(fakeSlack.Inbox)
				break WAIT
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	var inbox []Message
	for message := range fakeSlack.Inbox {
		inbox = append(inbox, message)
	}

	Equal(t, 2, len(inbox))
	Equal(t, "freshprince: Sup Jay!", inbox[0].String())
	Equal(t, "freshprince: Uncle Phil, you here?", inbox[1].String())
}

func TestSlack_WritePump(t *testing.T) {
	client := &fakeSlackAPI{}

	fakeSlack := &Slack{
		relayedChannelID: slackChatID,
		botUserID:        slackBotID,
		client:           client,
		Pump:             NewPump(),
	}

	fakeSlack.Outbox <- createTelegramMessage("Sup Jay!", "Will", "Smith", "freshprince")
	fakeSlack.Outbox <- createTelegramMessage(":clap: Psss!", "Will", "Smith", "freshprince")

	fakeSlack.WritePump()

	// wait for the pump to to process the channel up to 1 second, or timeout
	timeout := time.NewTimer(1 * time.Second)

WAIT:
	for {
		select {
		case <-timeout.C:
			Fail(t, "timeout while processing the Read Pump")
			break WAIT
		default:
			if len(fakeSlack.Outbox) == 0 {
				break WAIT
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	Equal(t, 2, len(client.sent))

	first := asSlackJSONMessage(client.sent[0])
	Equal(t, "Sup Jay!", first.Text)

	second := asSlackJSONMessage(client.sent[1])
	Equal(t, ":clap: Psss!", second.Text)
}

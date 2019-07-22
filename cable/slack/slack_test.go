package slack

import (
	"github.com/miguelff/cable/cable"
	api "github.com/nlopes/slack"
	. "github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSlack_GoRead(t *testing.T) {
	updates := []api.RTMEvent{
		createSlackUserReaction(),                                                                // discarded, unkown update
		createSlackBotMessage(slackChannelID, "Hey Hey!"),                                        // discarded, because written by the bot itself
		createSlackUserMessage(slackChannelID, slackUserID, "Sup Jay!"),                          // selected
		createSlackUserMessage(unknownSlackChannelID, slackUserID, "Uncle Phil, where are you?"), // discarded because written by a user in a chat other than the relayed channel
		createSlackUserMessage(slackChannelID, slackUserID, "Uncle Phil, you here?"),             // selected
		createSlackUserMessage(slackChannelID, unknownSlackUSerID, "Uncle Phil, you here?"),      // selected
		{}, // discarded: no message
	}

	updatesCh := make(chan api.RTMEvent, len(updates))
	for _, update := range updates {
		updatesCh <- update
	}

	userMap := make(UserMap)
	userMap[slackUserID] = createSlackUser(slackUserID, "Will Smith", "freshprince")

	fake := &Slack{
		relayedChannelID: slackChannelID,
		botUserID:        slackBotID,
		client: &fakeSlackAPI{
			rtmEvents: updatesCh,
			users:     userMap,
		},
	}
	fake.Pump = cable.NewPump(fake)

	fake.GoRead()

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
				close(fake.Inbox())
				break WAIT
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	fake.StopRead()

	var inbox []cable.Update
	for message := range fake.Inbox() {
		inbox = append(inbox, message)
	}

	Equal(t, 3, len(inbox))

	first := inbox[0].(cable.Message)
	Equal(t, "Sup Jay!", first.Contents.String())
	Equal(t, "freshprince", first.Author.String())

	second := inbox[1].(cable.Message)
	Equal(t, "Uncle Phil, you here?", second.Contents.String())
	Equal(t, "freshprince", second.Author.String())

	third := inbox[2].(cable.Message)
	Equal(t, "Uncle Phil, you here?", third.Contents.String())
	Equal(t, "UNKOWN_USER", third.Author.String())
}

func TestSlack_GoWrite(t *testing.T) {
	client := &fakeSlackAPI{}

	fake := &Slack{
		relayedChannelID: slackChannelID,
		botUserID:        slackBotID,
		client:           client,
	}
	fake.Pump = cable.NewPump(fake)

	fake.Outbox() <- cable.Message{
		Contents: &cable.Contents{
			Raw: "Sup Jay!",
		},
		Author: &cable.Author{
			Alias: "freshprince",
			Name:  "Will Smith",
		},
	}

	fake.Outbox() <- cable.Message{
		Contents: &cable.Contents{
			Raw: ":clap: Psss!",
		},
		Author: &cable.Author{
			Alias: "freshprince",
		},
	}

	fake.GoWrite()

	// wait for the pump to to process the channel up to 1 second, or timeout
	timeout := time.NewTimer(1 * time.Second)

WAIT:
	for {
		select {
		case <-timeout.C:
			Fail(t, "timeout while processing the Read Pump")
			break WAIT
		default:
			if len(fake.Outbox()) == 0 {
				break WAIT
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	fake.StopWrite()

	Equal(t, 2, len(client.sent))

	first := asSlackJSONMessage(client.sent[0])
	Equal(t, "Sup Jay!", first.Text)
	Equal(t, "Will Smith (freshprince)", first.AuthorName)

	second := asSlackJSONMessage(client.sent[1])
	Equal(t, ":clap: Psss!", second.Text)
	Equal(t, "freshprince", second.AuthorName)
}

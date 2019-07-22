package telegram

import (
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/miguelff/cable/cable"
	. "github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTelegram_GoRead(t *testing.T) {
	updates := []telegram.Update{
		createTelegramEdition(),
		createTelegramBotMessage(telegramChatID, "Hey Hey!"),                           // discarded, because written by the bot itself
		createTelegramUserMessage(telegramChatID, "Sup Jay!"),                          // selected
		createTelegramUserMessage(unknownTelegramChatID, "Uncle Phil, where are you?"), // discarded because written by a user in a chat other than the relayed channel
		createTelegramUserMessage(telegramChatID, "Uncle Phil, you here?"),             // selected
		{}, // discarded: no message
	}
	updatesCh := make(chan telegram.Update, len(updates))
	for _, update := range updates {
		updatesCh <- update
	}

	fake := &Telegram{
		relayedChatID: telegramChatID,
		botUserID:     telegramBotID,
		client:        &fakeTelegramAPI{updatesChannel: updatesCh},
		Pump:          cable.NewPump(),
	}

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

	Equal(t, 2, len(inbox))
	first := inbox[0].(cable.Message)
	Equal(t, "Sup Jay!", first.Contents.String())
	Equal(t, "freshprince", first.Author.String())

	second := inbox[1].(cable.Message)
	Equal(t, "Uncle Phil, you here?", second.Contents.String())
	Equal(t, "freshprince", second.Author.String())
}

func TestTelegram_GoWrite(t *testing.T) {
	client := &fakeTelegramAPI{}

	fake := &Telegram{
		relayedChatID: telegramChatID,
		botUserID:     telegramBotID,
		client:        client,
		Pump:          cable.NewPump(),
	}

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
	first := client.sent[0].(telegram.MessageConfig)
	Equal(t, "*Will Smith (freshprince):* Sup Jay!", first.Text)

	second := client.sent[1].(telegram.MessageConfig)
	Equal(t, "*freshprince:* 👏  Psss!", second.Text)
}

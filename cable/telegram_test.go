package cable

import (
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
	. "github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTelegram_ReadPump(t *testing.T) {
	updates := []telegram.Update{
		createTelegramBotUpdate(telegramChatID, "Hey Hey!"),                           // discarded, because written by the bot itself
		createTelegramUserUpdate(telegramChatID, "Sup Jay!"),                          // selected
		createTelegramUserUpdate(unknownTelegramChatID, "Uncle Phil, where are you?"), // discarded because written by a user in a chat other than the relayed channel
		createTelegramUserUpdate(telegramChatID, "Uncle Phil, you here?"),             // selected
		{}, // discarded: no message
	}
	updatesCh := make(chan telegram.Update, len(updates))
	for _, update := range updates {
		updatesCh <- update
	}

	fakeTelegram := &Telegram{
		relayedChatID: telegramChatID,
		botUserID:     telegramBotID,
		client:        &fakeTelegramAPI{updatesChannel: updatesCh},
		Pump:          NewPump(),
	}

	fakeTelegram.GoRead()

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
				close(fakeTelegram.InboxCh)
				break WAIT
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	fakeTelegram.StopRead()

	var inbox []Message
	for message := range fakeTelegram.InboxCh {
		inbox = append(inbox, message)
	}

	Equal(t, 2, len(inbox))
	Equal(t, "freshprince: Sup Jay!", inbox[0].String())
	Equal(t, "freshprince: Uncle Phil, you here?", inbox[1].String())
}

func TestTelegram_WritePump(t *testing.T) {
	client := &fakeTelegramAPI{}

	fakeTelegram := &Telegram{
		relayedChatID: telegramChatID,
		botUserID:     telegramBotID,
		client:        client,
		Pump:          NewPump(),
	}

	fakeTelegram.OutboxCh <- createSlackMessage("Sup Jay!", "WILL")
	fakeTelegram.OutboxCh <- createSlackMessage(":clap: Psss!", "JAZZ")

	fakeTelegram.GoWrite()

	// wait for the pump to to process the channel up to 1 second, or timeout
	timeout := time.NewTimer(1 * time.Second)

WAIT:
	for {
		select {
		case <-timeout.C:
			Fail(t, "timeout while processing the Read Pump")
			break WAIT
		default:
			if len(fakeTelegram.OutboxCh) == 0 {
				break WAIT
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	fakeTelegram.StopWrite()

	Equal(t, 2, len(client.sent))
	Equal(t, "*Stranger:* Sup Jay!", client.sent[0].(telegram.MessageConfig).Text)
	Equal(t, "*Stranger:* 👏  Psss!", client.sent[1].(telegram.MessageConfig).Text)
}

package cable

import (
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
	. "github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type botDouble struct {
	updatesChannel telegram.UpdatesChannel
	sent           []telegram.Chattable
}

func (b *botDouble) GetUpdatesChan(config telegram.UpdateConfig) (telegram.UpdatesChannel, error) {
	return b.updatesChannel, nil
}

func (b *botDouble) Send(c telegram.Chattable) (telegram.Message, error) {
	b.sent = append(b.sent, c)
	return telegram.Message{}, nil
}

const (
	botID = iota
	userID
	chatRoom
	unkownRoom
)

// createUpdate creates a message update as if it was written in the relayChannel,
// by a user with the given UserID, and with the given text
func createBotUpdate(relayedChannel int64, text string) telegram.Update {
	return telegram.Update{
		Message: &telegram.Message{
			Text: text,
			Chat: &telegram.Chat{
				ID: relayedChannel,
			},
			From: &telegram.User{
				ID:       botID,
				UserName: "CableBot",
			},
		},
	}
}

// createUpdate creates a message update as if it was written in the relayChannel,
// by a user with the given UserID, and with the given text
func createUserUpdate(relayedChannel int64, text string) telegram.Update {
	return telegram.Update{
		Message: &telegram.Message{
			Text: text,
			Chat: &telegram.Chat{
				ID: relayedChannel,
			},
			From: &telegram.User{
				ID:       userID,
				UserName: "freshprince",
			},
		},
	}
}

func TestTelegram_ReadPump(t *testing.T) {
	updates := []telegram.Update{
		{},                                     // message not set, discarded
		createBotUpdate(chatRoom, "Hey Hey!"),  // discarded, update by the bot
		createUserUpdate(chatRoom, "Sup Jay!"), // selected, by a user in the relayed channel
		createUserUpdate(unkownRoom, "Uncle Phil, where are you?"), // discarded, by a user in a chat other than the relayed channel
		createUserUpdate(chatRoom, "Uncle Phil, you here?"),        // discarded, by a user in a chat other than the relayed channel
	}
	updatesCh := make(chan telegram.Update, len(updates))
	for _, update := range updates {
		updatesCh <- update
	}

	fakeTelegram := &Telegram{
		relayedChannelID: chatRoom,
		botUserID:        botID,
		bot:              &botDouble{updatesChannel: updatesCh},
		Pump:             NewPump(),
	}

	fakeTelegram.ReadPump()

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
				close(fakeTelegram.Inbox)
				break WAIT
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	var inbox []Message
	for message := range fakeTelegram.Inbox {
		inbox = append(inbox, message)
	}

	Equal(t, 2, len(inbox))
	Equal(t, "freshprince: Sup Jay!", inbox[0].String())
	Equal(t, "freshprince: Uncle Phil, you here?", inbox[1].String())
}

func TestTelegram_WritePump(t *testing.T) {
	bot := &botDouble{}

	fakeTelegram := &Telegram{
		relayedChannelID: chatRoom,
		botUserID:        botID,
		bot:              bot,
		Pump:             NewPump(),
	}

	fakeTelegram.Outbox <- createSlackMessage("Sup Jay!", "WILL")
	fakeTelegram.Outbox <- createSlackMessage(":clap: Psss!", "JAZZ")

	fakeTelegram.WritePump()

	// wait for the pump to to process the channel up to 1 second, or timeout
	timeout := time.NewTimer(1 * time.Second)

WAIT:
	for {
		select {
		case <-timeout.C:
			Fail(t, "timeout while processing the Read Pump")
			break WAIT
		default:
			if len(fakeTelegram.Outbox) == 0 {
				break WAIT
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	Equal(t, 2, len(bot.sent))
	Equal(t, "*Stranger:* Sup Jay!", bot.sent[0].(telegram.MessageConfig).Text)
	Equal(t, "*Stranger:* 👏  Psss!", bot.sent[1].(telegram.MessageConfig).Text)
}

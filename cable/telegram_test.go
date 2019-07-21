package cable

import (
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
	. "github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type botDouble struct {
	updatesChannel telegram.UpdatesChannel
}

func (b *botDouble) GetUpdatesChan(config telegram.UpdateConfig) (telegram.UpdatesChannel, error) {
	return b.updatesChannel, nil
}

func (b *botDouble) Send(c telegram.Chattable) (telegram.Message, error) {
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

func TestSlack_ReadPump(t *testing.T) {
	updates := []telegram.Update{
		{},                                      // message not set, discarded
		createBotUpdate(chatRoom, "Hey Hey!"),   // discarded, update by the bot
		createUserUpdate(chatRoom, "Sup Jazz!"), // selected, by a user in the relayed channel
		createUserUpdate(unkownRoom, "Uncle Phil, where are you?"), // discarded, by a user in a chat other than the relayed channel
		createUserUpdate(chatRoom, "Uncle Phil, you here?"),        // discarded, by a user in a chat other than the relayed channel
	}
	updatesCh := make(chan telegram.Update, len(updates))
	for _, update := range updates {
		updatesCh <- update
	}

	fakeTelegram := &Telegram{
		relayedChannel: chatRoom,
		botUserID:      botID,
		bot:            &botDouble{updatesChannel: updatesCh},
		Pump:           NewPump(),
	}

	fakeTelegram.ReadPump()
	waitTilProcessed(t, updatesCh, fakeTelegram, time.Second)

	var inbox []Message
	for message := range fakeTelegram.Inbox {
		inbox = append(inbox, message)
	}

	Equal(t, 2, len(inbox))
	Equal(t, "freshprince: Sup Jazz!", inbox[0].String())
	Equal(t, "freshprince: Uncle Phil, you here?", inbox[1].String())
}

// waitTilProcessed waits until the updates channel is being processed by the read pump,
// or fails if it takes more to be processed than the given duration
func waitTilProcessed(t *testing.T, updatesCh chan telegram.Update, telegram *Telegram, duration time.Duration) {
	timeout := time.NewTimer(duration)
	for {
		select {
		case <-timeout.C:
			Fail(t, "timeout while processing the Read Pump")
			return
		default:
			if len(updatesCh) == 0 {
				close(telegram.Inbox)
				return
			}
			time.Sleep(1 * time.Millisecond)
		}
	}
}

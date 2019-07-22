package telegram

import (
	telegramAPI "github.com/go-telegram-bot-api/telegram-bot-api"
)

/* Constants used in tests */

const (
	telegramBotID = iota
	telegramUserID
	unknownTelegramUserID
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
func createTelegramBotMessage(relayedChannelID int64, text string) telegramAPI.Update {
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
func createTelegramUserMessage(relayedChannel int64, text string) telegramAPI.Update {
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

// createTelegramEdition creates a slackAPI.RTM update representing an edit
func createTelegramEdition() telegramAPI.Update {
	return telegramAPI.Update{
		EditedChannelPost: &telegramAPI.Message{},
	}
}

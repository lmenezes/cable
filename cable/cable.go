package cable

import (
	log "github.com/sirupsen/logrus"
)

// Setup configures the integration between slack and telegram. Creating a Pump
// where messages arrived to slack are written into telegram and vice-versa
func Setup(config *Config) {
	slack := NewSlack(config.SlackToken, config.SlackRelayedChannel, config.SlackBotUserID)
	slack.GoRead()
	slack.GoWrite()

	telegram := NewTelegram(config.TelegramToken, config.TelegramRelayedChannel, config.TelegramBotUserID, false)
	telegram.GoRead()
	telegram.GoWrite()

	NewBidirectionalPumpConnection(slack, telegram).Go()
	log.Infoln("Slack and Telegram are now connected.")
}

package cable

import (
	log "github.com/sirupsen/logrus"
)

// Setup configures the integration between slack and telegram. Creating a Pump
// where messages arrived to slack are written into telegram and vice-versa
func Setup(config *Config) {
	slack := NewSlack(config.SlackToken, config.SlackRelayedChannel, config.SlackBotUserID)
	telegram := NewTelegram(config.TelegramToken, config.TelegramRelayedChannel, config.TelegramBotUserID, false)
	NewBidirectionalPumpConnection(slack, telegram).Go()
	log.Infoln("Slack and Telegram are now connected.")
}

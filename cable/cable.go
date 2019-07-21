package cable

import (
	log "github.com/sirupsen/logrus"
)

// Setup configures the integration between slack and telegram. Creating a Pump
// where messages arrived to slack are written into telegram and vice-versa
func Setup(config *Config) {
	slack := NewSlack(config.SlackToken, config.SlackRelayedChannel, config.SlackBotUserID)
	slack.ReadPump()
	slack.WritePump()

	telegram := NewTelegram(config.TelegramToken, config.TelegramRelayedChannel, config.TelegramBotUserID)
	telegram.ReadPump()
	telegram.WritePump()

	go func() {
		for {
			select {
			case m := <-slack.Inbox:
				log.Debugln("[SLACK]", m)
				telegram.Outbox <- m
			case m := <-telegram.Inbox:
				log.Debugln("[TELEGRAM]", m)
				slack.Outbox <- m
			}
		}
	}()

	log.Infoln("Slack and Telegram are now connected.")
}

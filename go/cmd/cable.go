package main

import (
	"github.com/joho/godotenv"
	"github.com/miguelff/cable/go/cable"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func main() {
	log.SetLevel(log.DebugLevel)

	if err := godotenv.Load(); err != nil {
		log.Errorln(err)
	}

	config := cable.NewConfig()
	log.Debugf("Config %v", config)
	setupCable(config)

	http.HandleFunc("/_health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("OK"))
	})

	http.ListenAndServe(config.ListeningPort, nil)
}

// setupCable configures the integration between slack and telegram
func setupCable(config *cable.Config) {
	slack := cable.NewSlack(config.SlackToken, config.SlackRelayedChannel, config.SlackBotUserID)
	slack.ReadPump()
	slack.WritePump()

	telegram := cable.NewTelegram(config.TelegramToken, config.TelegramRelayedChannel, config.TelegramBotUserID)
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

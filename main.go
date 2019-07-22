package main

import (
	"github.com/joho/godotenv"
	"github.com/miguelff/cable/cable"
	s "github.com/miguelff/cable/cable/slack"
	t "github.com/miguelff/cable/cable/telegram"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func ok(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("OK"))
}

func main() {
	log.SetLevel(log.DebugLevel)

	if err := godotenv.Load(); err != nil {
		log.Info("Cannot load env file: ", err)
	}

	config := cable.NewConfig()
	log.Debugf("Config %v", config)

	slack := s.NewSlack(config.SlackToken, config.SlackRelayedChannel, config.SlackBotUserID)
	telegram := t.NewTelegram(config.TelegramToken, config.TelegramRelayedChannel, config.TelegramBotUserID, false)
	cable.NewBidirectionalPumpConnection(slack, telegram).Go()
	log.Infoln("Slack and Telegram are now connected.")

	http.HandleFunc("/_health", ok)
	http.HandleFunc("/", ok)

	_ = http.ListenAndServe(config.ListeningPort, nil)
}

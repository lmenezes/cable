package main

import (
	"fmt"
	"github.com/miguelff/cable/go/cable"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

func main() {
	log.SetLevel(log.DebugLevel)
	config := cable.NewConfig()
	setupCable(config)

	listeningPort := config.ListeningPort
	if !strings.HasPrefix(listeningPort, ":") {
		listeningPort = fmt.Sprintf(":%s", listeningPort)
	}

	http.HandleFunc("/_health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("OK"))
	})

	log.Debugf("Starting server in %s", listeningPort)
	http.ListenAndServe(listeningPort, nil)
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
				telegram.Outbox <- m
			case m := <-telegram.Inbox:
				slack.Outbox <- m
			}
		}
	}()
}

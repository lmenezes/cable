package main

import (
	"github.com/joho/godotenv"
	"github.com/miguelff/cable/cable"
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
	cable.Setup(config)

	http.HandleFunc("/_health", ok)
	http.HandleFunc("/", ok)

	_ = http.ListenAndServe(config.ListeningPort, nil)
}

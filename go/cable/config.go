package cable

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// Config struct containing the configuration for the application
type Config struct {
	ListeningPort          string
	SlackToken             string
	SlackRelayedChannel    string
	SlackBotUserID         string
	TelegramToken          string
	TelegramRelayedChannel int64
	TelegramBotUserID      int
}

// NewConfig creates a new value of Config
func NewConfig() *Config {
	listeningPort := getEnv("PORT")
	if !strings.HasPrefix(listeningPort, ":") {
		listeningPort = fmt.Sprintf(":%s", listeningPort)
	}

	return &Config{
		ListeningPort:          listeningPort,
		SlackToken:             getEnv("SLACK_TOKEN"),
		SlackRelayedChannel:    getEnv("SLACK_RELAYED_CHANNEL"),
		SlackBotUserID:         getEnv("SLACK_BOT_USER_ID"),
		TelegramToken:          getEnv("TELEGRAM_TOKEN"),
		TelegramRelayedChannel: int64(getEnvAsInt("TELEGRAM_RELAYED_CHANNEL")),
		TelegramBotUserID:      getEnvAsInt("TELEGRAM_BOT_USER_ID"),
	}
}

// Simple helper function to read an environment or return a default value
func getEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		log.Panicf("ENV VAR %s has to be set", key)
	}
	return value
}

// Simple helper function to read an environment variable into integer or return a default value
func getEnvAsInt(key string) int {
	valueStr := getEnv(key)
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Panicf("ENV VAR %s=%s cannot be converted to an integer", key, valueStr)
	}
	return value
}

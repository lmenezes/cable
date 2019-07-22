package cable

import (
	"fmt"
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
		TelegramRelayedChannel: getEnvAsInt64("TELEGRAM_RELAYED_CHANNEL"),
		TelegramBotUserID:      int(getEnvAsInt64("TELEGRAM_BOT_USER_ID")),
	}
}

// getEnv is a helper function to read an environment variable and panic if
// it is missing
func getEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		panic(fmt.Sprintf("ENV VAR %s has to be set", key))
	}
	return value
}

// getEnvAsInt64 is a helper function to read an environment variable as a int64
// and panic if it is missing or it cannot be parsed to int64
func getEnvAsInt64(key string) int64 {
	valueStr := getEnv(key)
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("ENV VAR %s=%s cannot be converted to an integer", key, valueStr))
	}
	return value
}

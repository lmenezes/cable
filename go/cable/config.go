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
	listeningPort := getEnv("PORT", ":8080")
	if !strings.HasPrefix(listeningPort, ":") {
		listeningPort = fmt.Sprintf(":%s", listeningPort)
	}

	return &Config{
		ListeningPort:          listeningPort,
		SlackToken:             getEnv("SLACK_TOKEN", ""),
		SlackRelayedChannel:    getEnv("SLACK_RELAYED_CHANNEL", "CKDQ4EQCR"), // telegram
		SlackBotUserID:         getEnv("SLACK_BOT_USER_ID", "BKPSC4XQR"),     // cable
		TelegramToken:          getEnv("TELEGRAM_TOKEN", ""),
		TelegramRelayedChannel: int64(getEnvAsInt("TELEGRAM_RELAYED_CHANNEL", -273875248)), // remote_ast
		TelegramBotUserID:      getEnvAsInt("TELEGRAM_BOT_USER_ID", 937923353),             // cable_telegram_bot
	}
}

// Simple helper function to read an environment or return a default value
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

// Simple helper function to read an environment variable into integer or return a default value
func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}

	return defaultVal
}

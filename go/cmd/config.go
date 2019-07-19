package main

import (
	"os"
	"strconv"
)

type Config struct {
	listeningPort          string
	slackToken             string
	slackRelayedChannel    string
	slackBotUserID         string
	telegramToken          string
	telegramRelayedChannel int64
	telegramBotUserID      int
}

func newConfig() *Config {
	return &Config{
		listeningPort:          getEnv("PORT", ":8080"),
		slackToken:             getEnv("SLACK_TOKEN", ""),
		slackRelayedChannel:    getEnv("SLACK_RELAYED_CHANNEL", "CKDQ4EQCR"), // telegram
		slackBotUserID:         getEnv("SLACK_BOT_USER_ID", "BKPSC4XQR"),     // cable
		telegramToken:          getEnv("TELEGRAM_TOKEN", ""),
		telegramRelayedChannel: int64(getEnvAsInt("TELEGRAM_RELAYED_CHANNEL", -273875248)), // remote_ast
		telegramBotUserID:      getEnvAsInt("TELEGRAM_BOT_USER_ID", 937923353),             // cable_telegram_bot
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

package cable

import (
	. "github.com/stretchr/testify/assert"
	"os"
	"testing"
)

var oldConfig = map[string]string{
	"SLACK_BOT_USER_ID":        os.Getenv("SLACK_BOT_USER_ID"),
	"SLACK_RELAYED_CHANNEL":    os.Getenv("SLACK_RELAYED_CHANNEL"),
	"SLACK_TOKEN":              os.Getenv("SLACK_TOKEN"),
	"TELEGRAM_BOT_USER_ID":     os.Getenv("TELEGRAM_BOT_USER_ID"),
	"TELEGRAM_RELAYED_CHANNEL": os.Getenv("TELEGRAM_RELAYED_CHANNEL"),
	"TELEGRAM_TOKEN":           os.Getenv("TELEGRAM_TOKEN"),
	"PORT":                     os.Getenv("PORT"),
}

var newConfig = map[string]string{
	"SLACK_BOT_USER_ID":        "YKKFA",
	"SLACK_RELAYED_CHANNEL":    "CLMKRRQRM",
	"SLACK_TOKEN":              "xoxb-XLuKt08ehunmReydHxSLEMAL",
	"TELEGRAM_BOT_USER_ID":     "9923353",
	"TELEGRAM_RELAYED_CHANNEL": "-3764886",
	"TELEGRAM_TOKEN":           "AAEm0DMVGVyzrr5xmKDITGKn51RNQ5j2nr0",
	"PORT":                     "8080",
}

func resetEnv() {
	for k, v := range oldConfig {
		os.Setenv(k, v)
	}
}

func setEnv() {
	for k, v := range newConfig {
		os.Setenv(k, v)
	}
}

func TestNewConfig_MissingConfigKey(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			Fail(t, "NewConfig() did not panicked when a config key is missing")
		}
		resetEnv()
	}()

	resetEnv()

	setEnv()
	os.Unsetenv("TELEGRAM_RELAYED_CHANNEL")
	NewConfig()
}

func TestNewConfig_WrongNumericKey(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			Fail(t, "NewConfig() did not panicked when a config key cannot be converted to an integer")
		}
		resetEnv()
	}()

	resetEnv()

	setEnv()
	os.Setenv("TELEGRAM_BOT_USER_ID", "NOT_AN_INTEGER")
	NewConfig()
}

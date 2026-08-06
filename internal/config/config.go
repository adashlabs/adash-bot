package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Token, DBPath, OpenAIBaseURL, OpenAIKey, OpenAIModel, OpenAISystemPrompt, SynapicKey string
	OpenAIMaxTokens                                                                      int
	OpenAITemperature                                                                    float64
	OpenAITimeout                                                                        time.Duration
}

func Load() Config {
	return Config{
		Token: os.Getenv("DISCORD_TOKEN"), DBPath: value("ADASH_DB_PATH", "data/adash.db"),
		OpenAIBaseURL: strings.TrimRight(os.Getenv("OPENAI_BASE_URL"), "/"), OpenAIKey: os.Getenv("OPENAI_API_KEY"),
		OpenAIModel: os.Getenv("OPENAI_MODEL"), OpenAISystemPrompt: os.Getenv("OPENAI_SYSTEM_PROMPT"), SynapicKey: os.Getenv("SYNAPIC_API_KEY"),
		OpenAIMaxTokens: integer("OPENAI_MAX_TOKENS", 900), OpenAITemperature: decimal("OPENAI_TEMPERATURE", .7),
		OpenAITimeout: time.Duration(integer("OPENAI_TIMEOUT_MS", 45000)) * time.Millisecond,
	}
}

func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func integer(key string, fallback int) int {
	v, e := strconv.Atoi(os.Getenv(key))
	if e == nil && v > 0 {
		return v
	}
	return fallback
}
func decimal(key string, fallback float64) float64 {
	v, e := strconv.ParseFloat(os.Getenv(key), 64)
	if e == nil && v > 0 {
		return v
	}
	return fallback
}

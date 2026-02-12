package config

import (
	"log"
	"os"
	"strconv"
)

type BotConfig struct {
	TelegramToken   string
	APIBaseURL      string
	AdminUser       string
	AdminPassword   string
	AdminTelegramID int64
	LogLevel        string
}

func MustLoad() BotConfig {
	cfg := BotConfig{
		TelegramToken: os.Getenv("TELEGRAM_TOKEN"),
		APIBaseURL:    os.Getenv("API_BASE_URL"),
		AdminUser:     os.Getenv("ADMIN_USER"),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		LogLevel:      os.Getenv("LOG_LEVEL"),
	}

	if cfg.TelegramToken == "" {
		log.Fatalf("TELEGRAM_TOKEN environment variable not set")
	}

	if cfg.APIBaseURL == "" {
		cfg.APIBaseURL = "https://api:8080"
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = "INFO"
	}

	if s := os.Getenv("ADMIN_TELEGRAM_ID"); s != "" {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			log.Fatalf("invalid ADMIN_TELEGRAM_ID %q: %v", s, err)
		}
		cfg.AdminTelegramID = id
	}

	return cfg
}

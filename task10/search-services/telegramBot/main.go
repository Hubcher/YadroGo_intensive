package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"yadro.com/course/telegramBot/adapters/rest"
	"yadro.com/course/telegramBot/adapters/tg"
	"yadro.com/course/telegramBot/config"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	cfg := config.MustLoad()

	log := mustMakeLogger(cfg.LogLevel)

	log.Info("starting telegram bot")

	apiClient := rest.NewClient(cfg.APIBaseURL, log, cfg.AdminUser, cfg.AdminPassword)

	bot, err := tg.NewBot(cfg.TelegramToken, apiClient, cfg.AdminTelegramID)

	if err != nil {
		log.Error("cannot create telegram bot", "error", err)
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Info("telegram bot started")
	if err := bot.Run(ctx); err != nil {
		log.Error("bot stopped with error", "error", err)
		return err
	}

	return nil
}

func mustMakeLogger(logLevel string) *slog.Logger {
	var level slog.Level
	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "ERROR":
		level = slog.LevelError
	default:
		panic("unknown log level: " + logLevel)
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level, AddSource: true})
	return slog.New(handler)
}

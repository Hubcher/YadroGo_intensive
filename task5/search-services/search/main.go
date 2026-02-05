package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/search/adapters/db"
	searchgrpc "yadro.com/course/search/adapters/grpc"
	"yadro.com/course/search/adapters/words"
	"yadro.com/course/search/config"
	"yadro.com/course/search/core"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	cfg := config.MustLoad(configPath)
	log := mustMakeLogger(cfg.LogLevel)

	log.Info("starting search server")
	log.Debug("debug messages are enabled")

	// DB
	storage, err := db.New(log, cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %v", err)
	}
	defer func() {
		if err := storage.Close(); err != nil {
			log.Error("failed to close db", "error", err)
		}
	}()

	// words
	wordsClient, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		return fmt.Errorf("failed create Words client: %v", err)
	}
	defer func() {
		if err := wordsClient.Close(); err != nil {
			log.Error("failed to close words client", "error", err)
		}
	}()

	// service
	searchService := core.NewService(log, storage, wordsClient)

	// gRPC server
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	searchpb.RegisterSearchServer(s, searchgrpc.NewServer(searchService))
	reflection.Register(s)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Debug("shutting down search server")
		s.GracefulStop()
	}()

	if err := s.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
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
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})
	return slog.New(handler)
}

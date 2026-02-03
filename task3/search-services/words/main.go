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
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	wordspb "yadro.com/course/proto/words"
	normalizer "yadro.com/course/words/words"
)

const (
	maxPhraseLen    = 4 << 10
	maxShutdownTime = 5 * time.Second
)

type Config struct {
	Port string `yaml:"port" env:"WORDS_GRPC_PORT" env-default:"8081"`
}
type server struct {
	wordspb.UnimplementedWordsServer
}

func (s *server) Ping(_ context.Context, in *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *server) Norm(ctx context.Context, req *wordspb.WordsRequest) (*wordspb.WordsReply, error) {

	phrase := req.GetPhrase()

	if len(phrase) > maxPhraseLen {
		return nil, status.Error(codes.ResourceExhausted, "phrase too large")
	}

	words := normalizer.Normalize(phrase)

	return &wordspb.WordsReply{Words: words}, nil

}
func main() {

	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "path to config file")
	flag.Parse()

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic(err)
	}

	log := slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		},
	))

	if err := run(cfg, log); err != nil {
		log.Error("failed to run", "error", err)
		os.Exit(1)
	}

}

func run(cfg Config, log *slog.Logger) error {
	addr := ":" + cfg.Port
	listener, err := net.Listen("tcp", addr)

	if err != nil {
		return fmt.Errorf("failed to listen port %s: %v", cfg.Port, err)
	}

	grpcServer := grpc.NewServer()
	wordspb.RegisterWordsServer(grpcServer, &server{})
	reflection.Register(grpcServer)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info("starting server", "addr", addr)
		if err = grpcServer.Serve(listener); err != nil {
			log.Error("failed to serve", "error", err)
			cancel()
		}
	}()

	<-ctx.Done()

	timer := time.AfterFunc(maxShutdownTime, func() {
		log.Info("forcing server stop")
		grpcServer.Stop()
	})
	defer timer.Stop()

	log.Info("starting graceful stop")
	grpcServer.GracefulStop()
	log.Info("server stopped")

	return err

}

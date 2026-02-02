package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ilyakaznacheev/cleanenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	wordspb "yadro.com/course/proto/words"
	normalizer "yadro.com/course/words/words"
)

const maxPhraseBytes = 4 << 10

type Config struct {
	WordsPort string `yaml:"port" env:"WORDS_GRPC_PORT" env-default:"8081"`
}

type server struct {
	wordspb.UnimplementedWordsServer
}

func (s *server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *server) Norm(ctx context.Context, req *wordspb.WordsRequest) (*wordspb.WordsReply, error) {

	phrase := req.GetPhrase()

	if len(phrase) > maxPhraseBytes {
		return nil, status.Error(codes.ResourceExhausted, "phrase too large")
	}

	words := normalizer.Normalize(phrase)

	slog.InfoContext(ctx, "normalized",
		"in_len", len(phrase),
		"out_count", len(words),
	)

	return &wordspb.WordsReply{Words: words}, nil
}

func main() {

	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "configuration file")
	flag.Parse()

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic(err)
	}

	addr := ":" + cfg.WordsPort
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("failed to listen", "addr", addr, "err", err)
		os.Exit(1)
	}

	grpcSrv := grpc.NewServer()
	wordspb.RegisterWordsServer(grpcSrv, &server{})
	reflection.Register(grpcSrv)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("words gRPC started", "addr", addr)
		if serveErr := grpcSrv.Serve(lis); serveErr != nil && ctx.Err() == nil {
			slog.Error("serve error", "err", serveErr)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down words gRPC...")
	grpcSrv.GracefulStop()
	slog.Info("server stopped")
}

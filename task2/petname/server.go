package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/ilyakaznacheev/cleanenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	petnamepb "yadro.com/course/proto"
)

type Config struct {
	Port string `yaml:"port" env:"PETNAME_GRPC_PORT" env-default:"8080"`
}

type server struct {
	petnamepb.UnimplementedPetnameGeneratorServer
}

func (s *server) Ping(_ context.Context, in *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *server) Generate(ctx context.Context, req *petnamepb.PetnameRequest) (*petnamepb.PetnameResponse, error) {

	words := req.GetWords() // в proto int64
	sep := req.GetSeparator()

	if words <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "words must be > 0")
	}

	// библиотека принимает именно int, а не int64
	name := petname.Generate(int(words), sep)
	slog.InfoContext(ctx, "generated", "name", name, "words", words, "separator", sep)

	return &petnamepb.PetnameResponse{Name: name}, nil
}

func (s *server) GenerateMany(req *petnamepb.PetnameStreamRequest, stream petnamepb.PetnameGenerator_GenerateManyServer) error {

	words := req.GetWords() // в proto int64
	sep := req.GetSeparator()
	names := req.GetNames() // в proto int64

	if words <= 0 {
		return status.Errorf(codes.InvalidArgument, "words must be > 0")
	}
	if names <= 0 {
		return status.Errorf(codes.InvalidArgument, "names must be > 0")
	}

	ctx := stream.Context()

	for i := int64(0); i < names; i++ {
		if err := ctx.Err(); err != nil {
			switch err {
			case context.Canceled:
				slog.Warn("context canceled")
				return status.Error(codes.Canceled, "client canceled")
			case context.DeadlineExceeded:
				slog.Warn("deadline exceeded")
				return status.Error(codes.DeadlineExceeded, "deadline exceeded")
			default:
				return status.Error(codes.Internal, "context error")
			}
		}

		name := petname.Generate(int(words), sep)

		if err := stream.Send(&petnamepb.PetnameResponse{Name: name}); err != nil {
			// сюда попадём, если клиент оборвал связь во время Send
			return err // gRPC сам проставит корректный код статуса
		}
	}
	return nil
}

func main() {

	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "configuration file")
	flag.Parse()

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic(err)
	}

	addr := ":" + cfg.Port
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("failed to listen", "addr", addr, "error", err)
		os.Exit(1)
	}

	grpcSrv := grpc.NewServer()
	petnamepb.RegisterPetnameGeneratorServer(grpcSrv, &server{})

	reflection.Register(grpcSrv)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Запуск сервера
	go func() {
		slog.Info("petname gRPC started", "addr", addr)
		if serveErr := grpcSrv.Serve(lis); serveErr != nil {
			if ctx.Err() == nil {
				slog.Error("serve error", "error", serveErr)
				os.Exit(1)
			}
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down gRPC server...")
	grpcSrv.GracefulStop()
	slog.Info("server stopped")
}

package grpc_test

import (
	"context"
	"io"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	pb "yadro.com/tests/proto/petname"
)

const petnameAddress = "localhost:28081"

func TestPetnamePreflight(t *testing.T) {
	require.Equal(t, true, true)
}

func TestGrpcPetnamePing(t *testing.T) {
	conn, err := grpc.NewClient(
		petnameAddress, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()
	c := pb.NewPetnameGeneratorClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = c.Ping(ctx, &emptypb.Empty{})
	require.NoError(t, err)
}

func TestGrpcPetname(t *testing.T) {
	conn, err := grpc.NewClient(
		petnameAddress, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()
	c := pb.NewPetnameGeneratorClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	separator := "_"
	words := rand.Int64N(10) + 2
	reply, err := c.Generate(ctx, &pb.PetnameRequest{Words: words, Separator: separator})
	require.NoError(t, err)
	name := reply.GetName()
	names := strings.Split(name, separator)
	require.Equal(t, words, int64(len(names)))

	reply, err = c.Generate(ctx, &pb.PetnameRequest{Words: words, Separator: separator})
	require.NoError(t, err)
	name2 := reply.GetName()
	require.NotEqual(t, name, name2)
	names = strings.Split(name, separator)
	require.Equal(t, words, int64(len(names)))
}

func TestGrpcPetnameNoWords(t *testing.T) {
	conn, err := grpc.NewClient(
		petnameAddress, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()
	c := pb.NewPetnameGeneratorClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	separator := "_"
	_, err = c.Generate(ctx, &pb.PetnameRequest{Words: 0, Separator: separator})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGrpcPetnameNegativeWords(t *testing.T) {
	conn, err := grpc.NewClient(
		petnameAddress, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()
	c := pb.NewPetnameGeneratorClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	separator := "_"
	_, err = c.Generate(ctx, &pb.PetnameRequest{Words: -1, Separator: separator})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGrpcPetnameStream(t *testing.T) {
	conn, err := grpc.NewClient(
		petnameAddress, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()
	c := pb.NewPetnameGeneratorClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	separator := "_"
	words := rand.Int64N(10) + 2
	total := rand.Int64N(100) + 20
	stream, err := c.GenerateMany(
		ctx, &pb.PetnameStreamRequest{Words: words, Separator: separator, Names: total})
	require.NoError(t, err)

	var count int64
	var previous string
	for {
		reply, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		name := reply.GetName()
		names := strings.Split(name, separator)
		require.Equal(t, words, int64(len(names)))
		require.NotEqual(t, previous, name)
		previous = name
		count++
	}
	require.Equal(t, total, count)
}

func TestGrpcPetnameStreamNoWords(t *testing.T) {
	conn, err := grpc.NewClient(
		petnameAddress, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()
	c := pb.NewPetnameGeneratorClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	separator := "_"
	total := rand.Int64N(100) + 20
	stream, err := c.GenerateMany(
		ctx, &pb.PetnameStreamRequest{Words: 0, Separator: separator, Names: total})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestGrpcPetnameStreamNegativeWords(t *testing.T) {
	conn, err := grpc.NewClient(
		petnameAddress, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()
	c := pb.NewPetnameGeneratorClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	separator := "_"
	total := rand.Int64N(100) + 20
	stream, err := c.GenerateMany(
		ctx, &pb.PetnameStreamRequest{Words: -1, Separator: separator, Names: total})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestGrpcPetnameStreamNoNames(t *testing.T) {
	conn, err := grpc.NewClient(
		petnameAddress, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()
	c := pb.NewPetnameGeneratorClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	separator := "_"
	words := rand.Int64N(100) + 20
	stream, err := c.GenerateMany(
		ctx, &pb.PetnameStreamRequest{Words: words, Separator: separator, Names: 0})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestGrpcPetnameStreamNegativeNames(t *testing.T) {
	conn, err := grpc.NewClient(
		petnameAddress, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()
	c := pb.NewPetnameGeneratorClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	separator := "_"
	words := rand.Int64N(100) + 20
	stream, err := c.GenerateMany(
		ctx, &pb.PetnameStreamRequest{Words: words, Separator: separator, Names: -1})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.Equal(t, status.Code(err), codes.InvalidArgument)
}

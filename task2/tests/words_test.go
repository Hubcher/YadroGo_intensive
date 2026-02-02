package grpc_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	pb "yadro.com/tests/proto/words"
)

const wordsAddress = "localhost:28082"

func TestWordsPreflight(t *testing.T) {
	require.Equal(t, true, true)
}

func TestGrpcWordsPing(t *testing.T) {
	conn, err := grpc.NewClient(
		wordsAddress, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()
	c := pb.NewWordsClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = c.Ping(ctx, &emptypb.Empty{})
	require.NoError(t, err)
}

func TestGrpcWords(t *testing.T) {
	conn, err := grpc.NewClient(
		wordsAddress, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()
	c := pb.NewWordsClient(conn)

	testCases := []struct {
		desc     string
		given    string
		expected []string
	}{
		{
			desc:     "empty",
			given:    "",
			expected: []string{},
		},
		{
			desc:     "simple",
			given:    "simple",
			expected: []string{"simpl"},
		},
		{
			desc:     "followers",
			given:    "I follow followers",
			expected: []string{"follow"},
		},
		{
			desc:     "punctuation",
			given:    "I shouted: 'give me your car!!!",
			expected: []string{"shout", "give", "car"},
		},
		{
			desc:     "stop words only",
			given:    "I and you or me or them, who will?",
			expected: []string{},
		},
		{
			desc:     "weird",
			given:    "Moscow!123'check-it'or   123, man,that,difficult:heck",
			expected: []string{"moscow", "check", "123", "man", "difficult", "heck"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			reply, err := c.Norm(ctx, &pb.WordsRequest{Phrase: tc.given})
			require.NoError(t, err)
			require.ElementsMatch(t, tc.expected, reply.GetWords())
		})
	}
}

func TestGrpcWordsTooLarge(t *testing.T) {
	conn, err := grpc.NewClient(
		wordsAddress, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()
	c := pb.NewWordsClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	phrase := strings.Repeat("1234", 1<<10)
	_, err = c.Norm(ctx, &pb.WordsRequest{Phrase: phrase})
	require.NoError(t, err)

	phrase += "0"
	_, err = c.Norm(ctx, &pb.WordsRequest{Phrase: phrase})
	require.Error(t, err)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
}

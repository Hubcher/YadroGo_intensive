package words

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"yadro.com/course/api/core"
	wordspb "yadro.com/course/proto/words"
)

type Client struct {
	log    *slog.Logger
	client wordspb.WordsClient
}

func NewClient(address string, log *slog.Logger) (*Client, error) {

	cc, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		return nil, err
	}

	// cc.Connect() // WithConnectParams - что то, что я искал, но не нашёл

	return &Client{
		log:    log,
		client: wordspb.NewWordsClient(cc),
	}, nil
}

func (c Client) Norm(ctx context.Context, phrase string) ([]string, error) {
	resp, err := c.client.Norm(ctx, &wordspb.WordsRequest{Phrase: phrase}, grpc.WaitForReady(true))
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.ResourceExhausted {
			return nil, core.ErrBadArguments
		}
		return nil, err
	}
	return resp.GetWords(), nil
}

func (c Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{}, grpc.WaitForReady(true))
	return err
}

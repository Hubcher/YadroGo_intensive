package search

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"yadro.com/course/api/core"
	searchpb "yadro.com/course/proto/search"
)

type Client struct {
	log    *slog.Logger
	client searchpb.SearchClient
	conn   *grpc.ClientConn
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  1 * time.Second,
				Multiplier: 1.6,
				MaxDelay:   10 * time.Second,
			},
			MinConnectTimeout: 10 * time.Second,
		}),
	)
	if err != nil {
		return nil, err
	}
	conn.Connect()

	return &Client{
		client: searchpb.NewSearchClient(conn),
		log:    log,
		conn:   conn,
	}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	return err
}

func (c *Client) Search(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {

	resp, err := c.client.Search(ctx, &searchpb.SearchRequest{
		Phrase: phrase,
		Limit:  int64(limit),
	})

	if err != nil {
		if status.Code(err) == codes.ResourceExhausted {
			return nil, core.ErrBadArguments
		}
		return nil, err
	}

	comics := make([]core.Comics, 0, len(resp.Comics))
	for _, cmt := range resp.Comics {
		comics = append(comics, core.Comics{
			ID:  int(cmt.Id),
			URL: cmt.Url,
		})
	}
	return comics, err

}

func (c *Client) Close() error {
	return c.conn.Close()
}

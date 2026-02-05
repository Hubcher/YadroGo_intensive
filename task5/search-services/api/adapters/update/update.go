package update

import (
	"context"
	"log/slog"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"yadro.com/course/api/core"
	updatepb "yadro.com/course/proto/update"
)

type Client struct {
	log    *slog.Logger
	client updatepb.UpdateClient
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
		client: updatepb.NewUpdateClient(conn),
		log:    log,
		conn:   conn,
	}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &empty.Empty{})
	return err

}

func (c *Client) Status(ctx context.Context) (core.UpdateStatus, error) {
	resp, err := c.client.Status(ctx, &emptypb.Empty{})
	if err != nil {
		return core.StatusUpdateUnknown, err
	}
	switch resp.GetStatus() {
	case updatepb.Status_STATUS_IDLE:
		return core.StatusUpdateIdle, nil
	case updatepb.Status_STATUS_RUNNING:
		return core.StatusUpdateRunning, nil
	default:
		return core.StatusUpdateUnknown, nil
	}
}

func (c *Client) Stats(ctx context.Context) (core.UpdateStats, error) {
	resp, err := c.client.Stats(ctx, &emptypb.Empty{})
	if err != nil {
		return core.UpdateStats{}, err
	}
	return core.UpdateStats{
		WordsTotal:    int(resp.GetWordsTotal()),
		WordsUnique:   int(resp.GetWordsUnique()),
		ComicsTotal:   int(resp.GetComicsTotal()),
		ComicsFetched: int(resp.GetComicsFetched()),
	}, nil
}

func (c *Client) Update(ctx context.Context) error {
	_, err := c.client.Update(ctx, &emptypb.Empty{})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return core.ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (c *Client) Drop(ctx context.Context) error {
	_, err := c.client.Drop(ctx, &emptypb.Empty{})
	return err
}

func (c *Client) Close() error {
	return c.conn.Close()
}

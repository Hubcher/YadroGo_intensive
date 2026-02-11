package events

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go"
)

const subjectDBUpdated = "xkcd.db.updated" // даем название топику, в который будем публиковать

type NatsPublisher struct {
	log *slog.Logger
	nc  *nats.Conn
}

func NewNatsPublisher(address string, log *slog.Logger) (*NatsPublisher, error) {
	nc, err := nats.Connect(address)
	if err != nil {
		return nil, err
	}
	log.Info("connected to broker for update", "address", address)

	return &NatsPublisher{
		log: log,
		nc:  nc,
	}, nil
}

func (p *NatsPublisher) NotifyDBChanged(ctx context.Context) error {

	if err := ctx.Err(); err != nil {
		return err
	}

	err := p.nc.Publish(subjectDBUpdated, []byte("XKCD DB has been updated"))
	if err != nil {
		return err
	}

	return p.nc.Flush()
}

func (p *NatsPublisher) Close() error {
	p.nc.Close()
	return nil
}

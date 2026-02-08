package events

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go"
)

const subjectDBUpdated = "xkcd.db.updated" // топик, в который публикует update service

type IndexerSearch interface {
	RebuildIndex(ctx context.Context) error
}
type NatsSubscriber struct {
	log *slog.Logger
	nc  *nats.Conn
	sub *nats.Subscription
}

func NewNatsSubscriber(address string, log *slog.Logger, indexerSearch IndexerSearch) (*NatsSubscriber, error) {

	nc, err := nats.Connect(address)
	if err != nil {
		return nil, err
	}

	log.Info("connected to broker for search", "address", address)

	s := &NatsSubscriber{
		log: log,
		nc:  nc,
	}

	subscribe, err := nc.Subscribe(subjectDBUpdated, func(msg *nats.Msg) {

		log.Info("received db update event", "subject", msg.Subject)
		// Перестраиваем индекс
		if err := indexerSearch.RebuildIndex(context.Background()); err != nil {
			log.Error("failed to rebuild index on event", "error", err)
		}
	})
	if err != nil {
		nc.Close()
		return nil, err
	}

	s.sub = subscribe
	return s, nil
}

func (s *NatsSubscriber) Close() error {
	if s.sub != nil {
		if err := s.sub.Unsubscribe(); err != nil {
			s.log.Error("failed to unsubscribe", "error", err)
		}
	}
	s.nc.Close()
	return nil
}

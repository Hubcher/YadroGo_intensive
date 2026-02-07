package indexer

import (
	"context"
	"log/slog"
	"time"

	"yadro.com/course/search/core"
)

func Run(ctx context.Context, log *slog.Logger, ttl time.Duration, indexer core.Indexer) {

	if err := indexer.RebuildIndex(ctx); err != nil {
		log.Error("initial index build failed", "error", err)
	} else {
		log.Info("initial index built")
	}

	ticker := time.NewTicker(ttl)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("stopping indexer")
			return
		case <-ticker.C:
			if err := indexer.RebuildIndex(ctx); err != nil {
				log.Error("index rebuild failed", "error", err)
			} else {
				log.Info("index rebuilt")
			}
		}
	}
}

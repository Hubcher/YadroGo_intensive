package core

import "context"

type DB interface {
	Search(ctx context.Context) ([]Comic, error)
}

type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}

type Searcher interface {
	Search(ctx context.Context, phrase string, limit int) ([]Comic, error)
}

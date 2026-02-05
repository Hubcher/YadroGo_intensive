package db

import (
	"context"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"yadro.com/course/search/core"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {
	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}
	return &DB{
		log:  log,
		conn: db,
	}, nil
}

func (db *DB) Search(ctx context.Context) ([]core.Comic, error) {
	type row struct {
		ID    int            `db:"id"`
		URL   string         `db:"url"`
		Words pq.StringArray `db:"words"`
	}

	var rows []row
	if err := db.conn.SelectContext(ctx, &rows, "SELECT id, url, words FROM comics"); err != nil {
		return nil, err
	}

	res := make([]core.Comic, 0, len(rows))
	for _, r := range rows {
		res = append(res, core.Comic{
			ID:    r.ID,
			URL:   r.URL,
			Words: r.Words,
		})
	}
	return res, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

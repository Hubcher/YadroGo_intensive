package db

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"yadro.com/course/update/core"
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

// isRetryable проверяет, относится ли ошибка postgreSQL к тем, что стоит повторять.
func isRetryable(err error, codes ...string) bool {
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return false
	}

	for _, c := range codes {
		if pg.Code == c {
			return true
		}
	}

	return false
}

// смак
func (db *DB) Add(ctx context.Context, comics core.Comics) error {

	// простая дедупликация слов внутри одного комикса
	if len(comics.Words) > 1 {
		seen := make(map[string]bool, len(comics.Words))
		uniq := make([]string, 0, len(comics.Words))
		for _, w := range comics.Words {
			if seen[w] {
				continue
			}
			seen[w] = true
			uniq = append(uniq, w)
		}
		sort.Strings(uniq)
		comics.Words = uniq
	}

	const maxAttempts = 5
	var lastErr error

	// одна попытка транзакции: возвращает err (nil — успех)
	runAttempt := func() error {

		tx, err := db.conn.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}

		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()

		// идемпотентно добавляем комикс
		_, err = tx.ExecContext(ctx, `
			insert into comics(id, img_url, title, alt)
			values ($1, $2, $3, $4)
			on conflict (id) do nothing`, // ничего не делаем
			comics.ID, comics.URL, comics.Title, comics.Description,
		)
		if err != nil {
			return err
		}

		// добавляем слова и связи
		for _, w := range comics.Words {

			var wordID int64

			// вставляем слова
			err = tx.GetContext(ctx, &wordID, `
				insert into words(word) values ($1)
				on conflict (word) do update set word = excluded.word
				returning id`, w,
			)
			if err != nil {
				return err
			}

			// вставляем связи
			_, err = tx.ExecContext(ctx, `
				insert into comic_words(comic_id, word_id)
				values ($1, $2)
				on conflict do nothing`,
				comics.ID, wordID,
			)
			if err != nil {
				return err
			}
		}

		return tx.Commit()
	}

	// ретраим попытки при дедлоке
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := runAttempt(); err != nil {
			// если две или более транзакции вставляют одни и те же слова, получается дедлок
			lastErr = err
			if isRetryable(err, "40P01") { // ошибка дедлока, взятая из логов бд
				// немного ждём и повторяем
				time.Sleep(time.Duration(attempt) * 150 * time.Millisecond)
				continue // следующая попытка
			}
			return err // не перезапускаемая ошибка
		}
		return nil // успех
	}
	return lastErr // исчерпали попытки
}

func (db *DB) Stats(ctx context.Context) (core.DBStats, error) {
	var st core.DBStats

	if err := db.conn.GetContext(ctx, &st.ComicsFetched, "select count(*) from comics"); err != nil {
		return core.DBStats{}, err
	}
	if err := db.conn.GetContext(ctx, &st.WordsUnique, "select count(*) from words"); err != nil {
		return core.DBStats{}, err
	}
	if err := db.conn.GetContext(ctx, &st.WordsTotal, "select count(*) from comic_words"); err != nil {
		return core.DBStats{}, err
	}
	return st, nil

}

func (db *DB) IDs(ctx context.Context) ([]int, error) {
	var ids []int
	if err := db.conn.SelectContext(ctx, &ids, "select id from comics order by id"); err != nil {
		return nil, err
	}
	return ids, nil
}

func (db *DB) Drop(ctx context.Context) error {
	// полная очистка
	_, err := db.conn.ExecContext(ctx, "TRUNCATE TABLE comic_words, words, comics restart identity cascade;")
	return err
}

package db

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sqlmock "github.com/zhashkevych/go-sqlxmock"
	"yadro.com/course/search/core"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newMockDB(t *testing.T) (*DB, sqlmock.Sqlmock) {
	t.Helper()

	conn, mock, err := sqlmock.Newx()
	require.NoError(t, err)

	return &DB{
		log:  newTestLogger(),
		conn: conn,
	}, mock
}

func TestDBSearch_Success(t *testing.T) {
	storage, mock := newMockDB(t)
	ctx := context.Background()

	// Настраиваем ожидаемый SELECT и возвращаем несколько строк
	rows := sqlmock.NewRows([]string{"id", "url", "words"}).
		AddRow(1, "u1", "{foo,bar}").
		AddRow(2, "u2", "{baz}")

	mock.ExpectQuery(`SELECT id, url, words FROM comics`).
		WithArgs(). // аргументов нет
		WillReturnRows(rows)

	result, err := storage.Search(ctx)
	require.NoError(t, err)
	require.Len(t, result, 2)

	// Проверяем первую строку
	assert.Equal(t, core.Comic{
		ID:    1,
		URL:   "u1",
		Words: []string{"foo", "bar"},
	}, result[0])

	// вторую тоже как бы не забываем
	assert.Equal(t, core.Comic{
		ID:    2,
		URL:   "u2",
		Words: []string{"baz"},
	}, result[1])

}

func TestDBSearch_QueryError(t *testing.T) {
	storage, mock := newMockDB(t)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT id, url, words FROM comics`).
		WithArgs().
		WillReturnError(assert.AnError)

	result, err := storage.Search(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
	assert.Nil(t, result)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDBClose(t *testing.T) {
	storage, mock := newMockDB(t)

	// Говорим мокy: ожидается вызов Close()
	mock.ExpectClose()

	err := storage.Close()
	require.NoError(t, err)

	// Проверяем, что все ожидания (включая Close) выполнены
	require.NoError(t, mock.ExpectationsWereMet())
}

package core

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Мокаем DB и Words
type mockDB struct {
	searchFn func(ctx context.Context) ([]Comic, error)
}

func (m *mockDB) Search(ctx context.Context) ([]Comic, error) {
	return m.searchFn(ctx)
}

type mockWords struct {
	normFn func(ctx context.Context, phrase string) ([]string, error)
}

func (m *mockWords) Norm(ctx context.Context, phrase string) ([]string, error) {
	return m.normFn(ctx, phrase)
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestService(t *testing.T, db DB, words Words) *Service {
	t.Helper()
	return NewService(newTestLogger(), db, words)
}

func TestServiceSearch_BadArguments(t *testing.T) {
	// готовим сервис с пустыми моками
	svc := newTestService(t,
		&mockDB{searchFn: func(ctx context.Context) ([]Comic, error) { return nil, nil }},
		&mockWords{normFn: func(ctx context.Context, phrase string) ([]string, error) { return nil, nil }},
	)

	ctx := context.Background()

	// таблица кейсов с некорректными аргументами
	testCases := []struct {
		name   string
		phrase string
		limit  int
	}{
		{"empty phrase", "", 1},           // пустая фраза
		{"non-positive limit", "foo", -1}, // limit <= 0
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := svc.Search(ctx, tc.phrase, tc.limit)

			// проверяем, что сервис возвращает ErrBadArguments (покрытие 34 строки сервиса)
			require.ErrorIs(t, err, ErrBadArguments)
			// при ошибке ожидаем nil-результат
			assert.Nil(t, got)
		})
	}
}

func TestServiceSearch_WordsError(t *testing.T) {
	// Покрытие 39 строки сервиса
	// Создаем ошибку от Words.Norm
	expErr := errors.New("norm failed")

	svc := newTestService(t,
		&mockDB{searchFn: func(ctx context.Context) ([]Comic, error) { return nil, nil }},
		// изменяем поведение мока Words, возвращаем ошибку
		&mockWords{normFn: func(ctx context.Context, phrase string) ([]string, error) {
			return nil, expErr
		}},
	)

	got, err := svc.Search(context.Background(), "foo", 10)
	// Метод Search должен просто пробросить ошибку наружу
	require.ErrorIs(t, err, expErr)
	// вернуть Nil
	assert.Nil(t, got)
}

func TestServiceSearch_NoWordsAfterNorm(t *testing.T) {
	// Покрытие 42 строки сервиса
	// Никаких ошибок, просто пустой слайс
	svc := newTestService(t,
		&mockDB{searchFn: func(ctx context.Context) ([]Comic, error) { return nil, nil }},
		&mockWords{normFn: func(ctx context.Context, phrase string) ([]string, error) {
			// имитируем, что после нормализации слов не осталось
			return []string{}, nil
		}},
	)

	res, err := svc.Search(context.Background(), "foo", 10)
	// Ошибок не должно быть
	require.NoError(t, err)
	// при отсутствии слов ожидаем nil-результат
	assert.Nil(t, res)
}

func TestServiceSearch_DBError(t *testing.T) {
	// Покрытие 52 строки сервиса
	expErr := errors.New("db error")

	svc := newTestService(t,
		&mockDB{searchFn: func(ctx context.Context) ([]Comic, error) {
			// Имитируем ошибку в бд
			return nil, expErr
		}},
		&mockWords{normFn: func(ctx context.Context, phrase string) ([]string, error) {
			// нормализация проходит успешно
			return []string{"foo"}, nil
		}},
	)

	res, err := svc.Search(context.Background(), "foo", 10)
	require.ErrorIs(t, err, expErr)
	assert.Nil(t, res)
}

func TestServiceSearch_ScoringAndLimit(t *testing.T) {
	// В этом тесте проверяем: 88, 91, 94, 99 строки сервиса
	// 1) корректный подсчёт совпадений и ratio;
	// 2) сортировку по matches, затем по ratio, затем по ID;
	// 3) ограничение по limit.

	// norm даёт два целевых слова
	words := &mockWords{normFn: func(ctx context.Context, phrase string) ([]string, error) {
		return []string{"foo", "bar"}, nil
	}}

	// БД возвращает нам несколько комиксов с разным числом совпадений и ratio
	db := &mockDB{searchFn: func(ctx context.Context) ([]Comic, error) {
		return []Comic{
			{ID: 1, URL: "u1", Words: []string{"foo", "baz"}},      // 1 совпадение, ratio 0.5
			{ID: 2, URL: "u2", Words: []string{"foo"}},             // 1 совпадение, ratio 1.0
			{ID: 3, URL: "u3", Words: []string{"foo", "bar"}},      // 2 совпадения, ratio 1.0 (топ)
			{ID: 4, URL: "u4", Words: nil},                         // пропускается (нет слов)
			{ID: 5, URL: "u5", Words: []string{"baz"}},             // 0 совпадений, пропуск
			{ID: 6, URL: "u6", Words: []string{"foo", "bar", "x"}}, // 2 совпадения, ratio 2/3
			{ID: 7, URL: "u7", Words: []string{"foo"}},             // копия ID 2, но с большим ID, по списку будет ниже
		}, nil
	}}

	svc := newTestService(t, db, words)

	// limit = 3 — ждём топ-3 результата
	res, err := svc.Search(context.Background(), "foo bar", 3)
	require.NoError(t, err)
	require.Len(t, res, 3)

	// Проверяем порядок по matches, потом по ratio, потом по ID
	// Ожидаем:
	// ID 3: matches=2, ratio=1.0
	// ID 6: matches=2, ratio=2/3
	// ID 2: matches=1, ratio=1.0 - именно его, а не c ID 7
	assert.Equal(t, 3, res[0].ID)
	assert.Equal(t, 6, res[1].ID)
	assert.Equal(t, 2, res[2].ID)
}

// Тесты для метода Service.RebuildIndex.
func TestServiceRebuildIndex_DBError(t *testing.T) {
	// 113 строка сервиса
	// Если db.Search возвращает ошибку — RebuildIndex должен её вернуть.
	expErr := errors.New("db error")
	svc := newTestService(t,
		&mockDB{searchFn: func(ctx context.Context) ([]Comic, error) { return nil, expErr }},
		&mockWords{normFn: func(ctx context.Context, phrase string) ([]string, error) { return nil, nil }},
	)

	err := svc.RebuildIndex(context.Background())
	require.ErrorIs(t, err, expErr)
}

func TestServiceRebuildIndex_Success(t *testing.T) {

	db := &mockDB{searchFn: func(ctx context.Context) ([]Comic, error) {
		return []Comic{
			{ID: 1, URL: "u1", Words: []string{"foo", "bar"}},
			{ID: 2, URL: "u2", Words: []string{"bar"}},
		}, nil
	}}

	svc := newTestService(t, db, &mockWords{
		normFn: func(ctx context.Context, phrase string) ([]string, error) { return nil, nil },
	})

	err := svc.RebuildIndex(context.Background())
	require.NoError(t, err)

	// проверяем, что индекс и карта комиксов заполнены
	require.Len(t, svc.comics, 2)

	// слово "bar" должно ссылаться на оба ID
	idsForBar := svc.index["bar"]
	require.Len(t, idsForBar, 2)
}

// Тесты для метода Service.IndexSearch.
func TestServiceIndexSearch_BadArguments(t *testing.T) {
	// 142 строка сервиса
	svc := newTestService(t,
		&mockDB{searchFn: func(ctx context.Context) ([]Comic, error) { return nil, nil }},
		&mockWords{normFn: func(ctx context.Context, phrase string) ([]string, error) { return nil, nil }},
	)

	ctx := context.Background()

	testCases := []struct {
		name   string
		phrase string
		limit  int
	}{
		{"empty phrase", "", 1},
		{"non-positive limit", "foo", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := svc.IndexSearch(ctx, tc.phrase, tc.limit)

			require.ErrorIs(t, err, ErrBadArguments)
			assert.Nil(t, res)
		})
	}
}

func TestServiceIndexSearch_WordsError(t *testing.T) {
	// 147 строка сервиса
	expErr := errors.New("norm error")

	svc := newTestService(t,
		&mockDB{searchFn: func(ctx context.Context) ([]Comic, error) { return nil, nil }},
		&mockWords{normFn: func(ctx context.Context, phrase string) ([]string, error) {
			return nil, expErr
		}},
	)

	res, err := svc.IndexSearch(context.Background(), "foo", 10)
	require.ErrorIs(t, err, expErr)
	assert.Nil(t, res)
}

func TestServiceIndexSearch_NoWordsAfterNorm(t *testing.T) {
	// 150 строка сервиса
	svc := newTestService(t,
		&mockDB{searchFn: func(ctx context.Context) ([]Comic, error) { return nil, nil }},
		&mockWords{normFn: func(ctx context.Context, phrase string) ([]string, error) {
			return []string{}, nil
		}},
	)

	res, err := svc.IndexSearch(context.Background(), "foo", 10)
	require.NoError(t, err)
	assert.Nil(t, res)
}

func TestServiceIndexSearch_EmptyIndexOrComics(t *testing.T) {
	// 157 строка сервиса
	// индекс ещё не построен, карты пустые - возвращаем nil без ошибок.
	svc := newTestService(t,
		&mockDB{searchFn: func(ctx context.Context) ([]Comic, error) { return nil, nil }},
		&mockWords{normFn: func(ctx context.Context, phrase string) ([]string, error) {
			return []string{"foo"}, nil
		}},
	)

	res, err := svc.IndexSearch(context.Background(), "foo", 10)
	require.NoError(t, err)
	assert.Nil(t, res)
}

func TestServiceIndexSearch_NoMatchesAfterIndex(t *testing.T) {
	// Покрытие 187 строки
	// Сценарий:
	// 1) строим индекс по слову "foo";
	// 2) ищем по слову "bar", которого в индексе нет;
	// 3) ожидаем nil-результат без ошибок.
	db := &mockDB{searchFn: func(ctx context.Context) ([]Comic, error) {
		return []Comic{
			{ID: 1, URL: "u1", Words: []string{"foo"}},
		}, nil
	}}

	words := &mockWords{normFn: func(ctx context.Context, phrase string) ([]string, error) {
		return []string{"bar"}, nil
	}}

	svc := newTestService(t, db, words)

	err := svc.RebuildIndex(context.Background())
	require.NoError(t, err)

	res, err := svc.IndexSearch(context.Background(), "bar", 10)
	require.NoError(t, err)
	assert.Nil(t, res)
}

func TestServiceIndexSearch_ScoringAndLimit(t *testing.T) {
	// Аналогичный тест для IndexSearch:
	// проверяем сортировку, подсчёт matches/ratio и limit.
	db := &mockDB{searchFn: func(ctx context.Context) ([]Comic, error) {
		return []Comic{
			{ID: 1, URL: "u1", Words: []string{"foo", "baz"}},      // 1 совпадение, ratio 0.5
			{ID: 2, URL: "u2", Words: []string{"foo"}},             // 1 совпадение, ratio 1.0
			{ID: 3, URL: "u3", Words: []string{"foo", "bar"}},      // 2 совпадения, ratio 1.0 (топ)
			{ID: 4, URL: "u4", Words: nil},                         // пропускается (нет слов)
			{ID: 5, URL: "u5", Words: []string{"baz"}},             // 0 совпадений, пропуск
			{ID: 6, URL: "u6", Words: []string{"foo", "bar", "x"}}, // 2 совпадения, ratio 2/3
			{ID: 7, URL: "u7", Words: []string{"foo"}},             // копия ID 2, но с большим ID, по списку будет ниже
		}, nil
	}}

	words := &mockWords{normFn: func(ctx context.Context, phrase string) ([]string, error) {
		return []string{"foo", "bar"}, nil
	}}

	svc := newTestService(t, db, words)

	err := svc.RebuildIndex(context.Background())
	require.NoError(t, err)

	res, err := svc.IndexSearch(context.Background(), "foo bar", 3)
	require.NoError(t, err)
	require.Len(t, res, 3)

	// Проверяем порядок по matches, потом по ratio, потом по ID
	// Ожидаем:
	// ID 3: matches=2, ratio=1.0
	// ID 6: matches=2, ratio=2/3
	// ID 2: matches=1, ratio=1.0 - именно его, а не c ID 7
	assert.Equal(t, 3, res[0].ID)
	assert.Equal(t, 6, res[1].ID)
	assert.Equal(t, 2, res[2].ID)
}

func TestServiceIndexSearch_SkipZeroWordCount(t *testing.T) {
	// Ручками заполняем индекс так, чтобы один из комиксов
	// попал в byId, но у него Words = nil
	// Этот комикс должен быть пропущен при подсчёте ratio и формировании результата.
	svc := &Service{
		db: nil, // в этом тесте БД не используется
		words: &mockWords{normFn: func(ctx context.Context, phrase string) ([]string, error) {
			return []string{"foo"}, nil
		}},
		index: map[string][]int{
			"foo": {1, 2}, // оба ID связаны со словом foo
		},
		comics: map[int]Comic{
			1: {ID: 1, URL: "u1", Words: nil},             // будет пропущен по wordCount == 0
			2: {ID: 2, URL: "u2", Words: []string{"foo"}}, // останется в выдаче
		},
	}

	res, err := svc.IndexSearch(context.Background(), "foo", 10)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, 2, res[0].ID)
}

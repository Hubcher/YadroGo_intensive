package core

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type mockDB struct {
	addFn   func(ctx context.Context, c Comics) error
	statsFn func(ctx context.Context) (DBStats, error)
	dropFn  func(ctx context.Context) error
	idsFn   func(ctx context.Context) ([]int, error)
}

func (m *mockDB) Add(ctx context.Context, c Comics) error {
	if m.addFn == nil {
		return nil
	}
	return m.addFn(ctx, c)
}

func (m *mockDB) Stats(ctx context.Context) (DBStats, error) {
	if m.statsFn == nil {
		return DBStats{}, nil
	}
	return m.statsFn(ctx)
}

func (m *mockDB) Drop(ctx context.Context) error {
	if m.dropFn == nil {
		return nil
	}
	return m.dropFn(ctx)
}

func (m *mockDB) IDs(ctx context.Context) ([]int, error) {
	if m.idsFn == nil {
		return nil, nil
	}
	return m.idsFn(ctx)
}

type mockXKCD struct {
	getFn    func(ctx context.Context, id int) (XKCDInfo, error)
	lastIDFn func(ctx context.Context) (int, error)
}

func (m *mockXKCD) Get(ctx context.Context, id int) (XKCDInfo, error) {
	if m.getFn == nil {
		return XKCDInfo{}, nil
	}
	return m.getFn(ctx, id)
}

func (m *mockXKCD) LastID(ctx context.Context) (int, error) {
	if m.lastIDFn == nil {
		return 0, nil
	}
	return m.lastIDFn(ctx)
}

type mockWords struct {
	normFn func(ctx context.Context, phrase string) ([]string, error)
}

func (m *mockWords) Norm(ctx context.Context, phrase string) ([]string, error) {
	return m.normFn(ctx, phrase)
}

type mockEvents struct {
	notifyFn func(ctx context.Context) error
}

func (m *mockEvents) NotifyDBChanged(ctx context.Context) error {
	if m.notifyFn == nil {
		return nil
	}
	return m.notifyFn(ctx)
}

func newUpdateService(
	t *testing.T,
	db DB,
	xkcd XKCD,
	words Words,
	concurrency int,
	events EventPublisher,
) *Service {
	t.Helper()
	svc, err := NewService(newTestLogger(), db, xkcd, words, concurrency, events)
	require.NoError(t, err)
	require.NotNil(t, svc)
	return svc
}

func TestNewService_BadConcurrency(t *testing.T) {
	// Покрытие 31 строки сервиса
	svc, err := NewService(
		newTestLogger(),
		&mockDB{},
		&mockXKCD{},
		&mockWords{},
		0,
		&mockEvents{},
	)

	require.Error(t, err)
	assert.Nil(t, svc)
}

func TestNewService_OK(t *testing.T) {
	svc, err := NewService(
		newTestLogger(),
		&mockDB{},
		&mockXKCD{},
		&mockWords{},
		3,
		&mockEvents{},
	)

	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Equal(t, 3, svc.concurrency)
}

func TestServiceUpdate_AlreadyRunning(t *testing.T) {
	// Покрытие 85 строки сервиса
	// Проверяем, что при повторном запуске возвращается ErrAlreadyExists.
	db := &mockDB{
		idsFn: func(ctx context.Context) ([]int, error) {
			return nil, nil
		},
	}
	xkcd := &mockXKCD{
		lastIDFn: func(ctx context.Context) (int, error) {
			return 0, nil
		},
	}

	svc := newUpdateService(t, db, xkcd, &mockWords{}, 1, &mockEvents{})

	// имитируем, что уже выполняется другая Update
	svc.running.Store(true)

	err := svc.Update(context.Background())
	require.ErrorIs(t, err, ErrAlreadyExists)
}

func TestServiceUpdate_LastIDError(t *testing.T) {
	// Покрытие 91 строки
	expErr := errors.New("last id failed")

	db := &mockDB{
		idsFn: func(ctx context.Context) ([]int, error) {
			return nil, nil
		},
	}
	xkcd := &mockXKCD{
		lastIDFn: func(ctx context.Context) (int, error) {
			return 0, expErr
		},
	}

	svc := newUpdateService(t, db, xkcd, &mockWords{}, 1, &mockEvents{})

	err := svc.Update(context.Background())
	require.ErrorIs(t, err, expErr)
}

func TestServiceUpdate_DBIDsError(t *testing.T) {
	// Покрытие 97 строки сервиса
	expErr := errors.New("ids failed")

	db := &mockDB{
		idsFn: func(ctx context.Context) ([]int, error) {
			return nil, expErr
		},
	}
	xkcd := &mockXKCD{
		lastIDFn: func(ctx context.Context) (int, error) {
			return 10, nil
		},
	}

	svc := newUpdateService(t, db, xkcd, &mockWords{}, 1, &mockEvents{})

	err := svc.Update(context.Background())
	require.ErrorIs(t, err, expErr)
}

func TestServiceUpdate_NoMissing_Comics(t *testing.T) {
	// Ситуация: все ID уже есть в БД -> missing пустой -> NotifyDBChanged не вызывается.
	db := &mockDB{
		idsFn: func(ctx context.Context) ([]int, error) {
			return []int{1, 2, 3}, nil
		},
	}
	xkcd := &mockXKCD{
		lastIDFn: func(ctx context.Context) (int, error) {
			return 3, nil
		},
	}
	notifyCalls := 0
	events := &mockEvents{
		notifyFn: func(ctx context.Context) error {
			notifyCalls++
			return nil
		},
	}

	// любые вызовы Get/Norm/Add здесь будут ошибкой - их быть не должно
	db.addFn = func(ctx context.Context, c Comics) error {
		t.Fatalf("db.Add should not be called when no missing comics")
		return nil
	}
	xkcd.getFn = func(ctx context.Context, id int) (XKCDInfo, error) {
		t.Fatalf("xkcd.Get should not be called when no missing comics")
		return XKCDInfo{}, nil
	}

	svc := newUpdateService(t, db, xkcd, &mockWords{}, 2, events)

	err := svc.Update(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, notifyCalls)
}

func TestServiceUpdate_Success_Simple(t *testing.T) {
	// 148 строка сервиса
	// last = 5, в БД есть [2,3], значит нужно скачать [1,4,5].
	var mu sync.Mutex
	var fetchedIDs []int
	var addedIDs []int

	db := &mockDB{
		idsFn: func(ctx context.Context) ([]int, error) {
			return []int{2, 3}, nil
		},
		addFn: func(ctx context.Context, c Comics) error {
			mu.Lock()
			defer mu.Unlock()
			addedIDs = append(addedIDs, c.ID)
			return nil
		},
	}

	xkcd := &mockXKCD{
		lastIDFn: func(ctx context.Context) (int, error) {
			return 5, nil
		},
		getFn: func(ctx context.Context, id int) (XKCDInfo, error) {
			mu.Lock()
			fetchedIDs = append(fetchedIDs, id)
			mu.Unlock()
			return XKCDInfo{
				ID:          id,
				URL:         "http://example.com",
				Title:       "title",
				Description: "desc",
			}, nil
		},
	}

	words := &mockWords{
		normFn: func(ctx context.Context, phrase string) ([]string, error) {
			// нормализатор пусть всегда возвращает один "условный" токен
			return []string{"token"}, nil
		},
	}

	notifyCalls := 0
	events := &mockEvents{
		notifyFn: func(ctx context.Context) error {
			notifyCalls++
			return nil
		},
	}

	svc := newUpdateService(t, db, xkcd, words, 2, events)

	err := svc.Update(context.Background())
	require.NoError(t, err)

	// Проверяем, какие ID были запрошены у xkcd и добавлены в БД.
	assert.ElementsMatch(t, []int{1, 4, 5}, fetchedIDs)
	assert.ElementsMatch(t, []int{1, 4, 5}, addedIDs)

	// Уведомление о смене БД должно быть отправлено один раз
	assert.Equal(t, 1, notifyCalls)
}

func TestServiceUpdate_Skip404(t *testing.T) {
	// покрытие 111 строки сервиса
	// Проверяем логику: ID 404 пропускается.
	// last >= 404, в БД ничего нет, значит цикл идёт по [1..last],
	// но на 404 должен сделать continue без добавления в missing.
	db := &mockDB{
		idsFn: func(ctx context.Context) ([]int, error) {
			return nil, nil
		},
		addFn: func(ctx context.Context, c Comics) error {
			// просто ничего, нам важно, какие ID дошли до Add
			if c.ID == 404 {
				t.Fatalf("ID 404 should be skipped and never added")
			}
			return nil
		},
	}

	xkcd := &mockXKCD{
		lastIDFn: func(ctx context.Context) (int, error) {
			return 405, nil
		},
		getFn: func(ctx context.Context, id int) (XKCDInfo, error) {
			if id == 404 {
				t.Fatalf("ID 404 should be skipped and never requested")
			}
			return XKCDInfo{
				ID:          id,
				URL:         "url",
				Title:       "t",
				Description: "d",
			}, nil
		},
	}

	words := &mockWords{
		normFn: func(ctx context.Context, phrase string) ([]string, error) {
			return []string{"w"}, nil
		},
	}

	svc := newUpdateService(t, db, xkcd, words, 1, &mockEvents{})

	err := svc.Update(context.Background())
	require.NoError(t, err)
}

func TestServiceUpdate_CancelledContext(t *testing.T) {
	// покрытие 134 строки
	// ctx.Done возвращает ctx.Err, а NotifyDBChanged не вызывается.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	db := &mockDB{
		idsFn: func(ctx context.Context) ([]int, error) {
			return nil, nil
		},
	}
	xkcd := &mockXKCD{
		lastIDFn: func(ctx context.Context) (int, error) {
			return 5, nil
		},
	}
	notifyCalls := 0
	events := &mockEvents{
		notifyFn: func(ctx context.Context) error {
			notifyCalls++
			return nil
		},
	}

	svc := newUpdateService(t, db, xkcd, &mockWords{}, 1, events)

	err := svc.Update(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, notifyCalls)
}

func TestServiceUpdate_NotifyDBChangedError(t *testing.T) {
	// Покрытие 145 строки сервиса
	expErr := errors.New("notify failed")

	db := &mockDB{
		idsFn: func(ctx context.Context) ([]int, error) {
			return nil, nil
		},
		addFn: func(ctx context.Context, c Comics) error {
			return nil
		},
	}

	xkcd := &mockXKCD{
		// пусть есть всего один комикс
		lastIDFn: func(ctx context.Context) (int, error) {
			return 1, nil
		},
		getFn: func(ctx context.Context, id int) (XKCDInfo, error) {
			// возвращаем валидную информацию, чтобы worker дошёл до Add
			return XKCDInfo{
				ID:          id,
				URL:         "http://example.com",
				Title:       "title",
				Description: "desc",
			}, nil
		},
	}

	words := &mockWords{
		// нормализация успешна, чтобы ничего не упало по пути
		normFn: func(ctx context.Context, phrase string) ([]string, error) {
			return []string{"token"}, nil
		},
	}

	// падение: "failed to send db-changed event"
	events := &mockEvents{
		notifyFn: func(ctx context.Context) error {
			return expErr
		},
	}

	svc := newUpdateService(t, db, xkcd, words, 1, events)

	err := svc.Update(context.Background())
	require.ErrorIs(t, err, expErr)
}

func TestServiceWorker_XKCDError(t *testing.T) {
	// покрытие 58 строки сервиса
	// xkcd.Get возвращает ошибку -> Norm и Add вызываться не должны.
	db := &mockDB{
		addFn: func(ctx context.Context, c Comics) error {
			t.Fatalf("db.Add should not be called when xkcd.Get fails")
			return nil
		},
	}
	xkcd := &mockXKCD{
		getFn: func(ctx context.Context, id int) (XKCDInfo, error) {
			return XKCDInfo{}, errors.New("xkcd failed")
		},
	}
	words := &mockWords{
		normFn: func(ctx context.Context, phrase string) ([]string, error) {
			t.Fatalf("Words.Norm should not be called when xkcd.Get fails")
			return nil, nil
		},
	}

	svc := &Service{
		log:   newTestLogger(),
		db:    db,
		xkcd:  xkcd,
		words: words,
	}

	jobs := make(chan int, 1)
	jobs <- 1
	close(jobs)

	svc.worker(context.Background(), jobs)
}

func TestServiceWorker_WordsError(t *testing.T) {
	// покрытие 65 строки сервиса
	// xkcd.Get ок, Words.Norm возвращает ошибку -> db.Add не вызывается.
	db := &mockDB{
		addFn: func(ctx context.Context, c Comics) error {
			t.Fatalf("db.Add should not be called when Words.Norm fails")
			return nil
		},
	}
	xkcd := &mockXKCD{
		getFn: func(ctx context.Context, id int) (XKCDInfo, error) {
			return XKCDInfo{
				ID:          id,
				URL:         "url",
				Title:       "t",
				Description: "d",
			}, nil
		},
	}
	words := &mockWords{
		normFn: func(ctx context.Context, phrase string) ([]string, error) {
			return nil, errors.New("norm failed")
		},
	}

	svc := &Service{
		log:   newTestLogger(),
		db:    db,
		xkcd:  xkcd,
		words: words,
	}

	jobs := make(chan int, 1)
	jobs <- 1
	close(jobs)

	svc.worker(context.Background(), jobs)
}

func TestServiceWorker_DBError(t *testing.T) {
	// Покрытие 78 строки сервиса
	// xkcd.Get и Norm ок, db.Add возвращает ошибку — должен просто залогироваться,
	// наружу ошибка не уходит.
	dbCalls := 0
	db := &mockDB{
		addFn: func(ctx context.Context, c Comics) error {
			dbCalls++
			return errors.New("db add failed")
		},
	}
	xkcd := &mockXKCD{
		getFn: func(ctx context.Context, id int) (XKCDInfo, error) {
			return XKCDInfo{
				ID:          id,
				URL:         "url",
				Title:       "t",
				Description: "d",
			}, nil
		},
	}
	words := &mockWords{
		normFn: func(ctx context.Context, phrase string) ([]string, error) {
			return []string{"w"}, nil
		},
	}

	svc := &Service{
		log:   newTestLogger(),
		db:    db,
		xkcd:  xkcd,
		words: words,
	}

	jobs := make(chan int, 1)
	jobs <- 1
	close(jobs)

	svc.worker(context.Background(), jobs)
	assert.Equal(t, 1, dbCalls)
}

func TestServiceStats_DBError(t *testing.T) {
	// 155 строка сервиса
	expErr := errors.New("stats failed")
	db := &mockDB{
		statsFn: func(ctx context.Context) (DBStats, error) {
			return DBStats{}, expErr
		},
	}
	xkcd := &mockXKCD{
		lastIDFn: func(ctx context.Context) (int, error) {
			t.Fatalf("LastID should not be called when Stats fails")
			return 0, nil
		},
	}

	svc := newUpdateService(t, db, xkcd, &mockWords{}, 1, &mockEvents{})

	_, err := svc.Stats(context.Background())
	require.ErrorIs(t, err, expErr)
}

func TestServiceStats_LastIDError(t *testing.T) {
	// покрытие 159 строки сервиса
	expErr := errors.New("last id failed")

	db := &mockDB{
		statsFn: func(ctx context.Context) (DBStats, error) {
			return DBStats{WordsTotal: 10}, nil
		},
	}
	xkcd := &mockXKCD{
		lastIDFn: func(ctx context.Context) (int, error) {
			return 0, expErr
		},
	}

	svc := newUpdateService(t, db, xkcd, &mockWords{}, 1, &mockEvents{})

	_, err := svc.Stats(context.Background())
	require.ErrorIs(t, err, expErr)
}

func TestServiceStats_ComicsTotal_NoHole(t *testing.T) {
	db := &mockDB{
		statsFn: func(ctx context.Context) (DBStats, error) {
			return DBStats{
				WordsTotal:    100,
				WordsUnique:   20,
				ComicsFetched: 10,
			}, nil
		},
	}
	xkcd := &mockXKCD{
		lastIDFn: func(ctx context.Context) (int, error) {
			return 10, nil // не дошли до 404 -> дыр нет
		},
	}

	svc := newUpdateService(t, db, xkcd, &mockWords{}, 1, &mockEvents{})

	stats, err := svc.Stats(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 10, stats.ComicsTotal)
	assert.Equal(t, 100, stats.WordsTotal)
	assert.Equal(t, 20, stats.WordsUnique)
	assert.Equal(t, 10, stats.ComicsFetched)
}

func TestServiceStats_ComicsTotal_WithHole(t *testing.T) {
	// last >= 404 -> считаем одну дыру и вычитаем её из общего количества.
	db := &mockDB{
		statsFn: func(ctx context.Context) (DBStats, error) {
			return DBStats{
				WordsTotal:    200,
				WordsUnique:   40,
				ComicsFetched: 50,
			}, nil
		},
	}
	xkcd := &mockXKCD{
		lastIDFn: func(ctx context.Context) (int, error) {
			return 405, nil
		},
	}

	svc := newUpdateService(t, db, xkcd, &mockWords{}, 1, &mockEvents{})

	stats, err := svc.Stats(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 404, stats.ComicsTotal) // 405 - 1 дыра
}

func TestServiceStatus(t *testing.T) {
	svc := newUpdateService(t, &mockDB{}, &mockXKCD{}, &mockWords{}, 1, &mockEvents{})

	// по умолчанию running=false -> StatusIdle
	assert.Equal(t, StatusIdle, svc.Status(context.Background()))

	// имитируем запущенный процесс
	svc.running.Store(true)
	assert.Equal(t, StatusRunning, svc.Status(context.Background()))
}

func TestServiceDrop_DBError(t *testing.T) {
	// 183
	expErr := errors.New("drop failed")

	db := &mockDB{
		dropFn: func(ctx context.Context) error {
			return expErr
		},
	}
	events := &mockEvents{
		notifyFn: func(ctx context.Context) error {
			t.Fatalf("NotifyDBChanged should not be called when Drop fails")
			return nil
		},
	}

	svc := newUpdateService(t, db, &mockXKCD{}, &mockWords{}, 1, events)

	err := svc.Drop(context.Background())
	require.ErrorIs(t, err, expErr)
}

func TestServiceDrop_NotifyErrorIgnored(t *testing.T) {
	expErr := errors.New("notify failed")

	db := &mockDB{
		dropFn: func(ctx context.Context) error {
			return nil
		},
	}
	notifyCalls := 0
	events := &mockEvents{
		notifyFn: func(ctx context.Context) error {
			notifyCalls++
			return expErr
		},
	}

	svc := newUpdateService(t, db, &mockXKCD{}, &mockWords{}, 1, events)

	err := svc.Drop(context.Background())
	require.ErrorIs(t, err, expErr)
	assert.Equal(t, 1, notifyCalls)
}

func TestServiceDrop_Success(t *testing.T) {
	db := &mockDB{
		dropFn: func(ctx context.Context) error {
			return nil
		},
	}
	notifyCalls := 0
	events := &mockEvents{
		notifyFn: func(ctx context.Context) error {
			notifyCalls++
			return nil
		},
	}

	svc := newUpdateService(t, db, &mockXKCD{}, &mockWords{}, 1, events)

	err := svc.Drop(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, notifyCalls)
}

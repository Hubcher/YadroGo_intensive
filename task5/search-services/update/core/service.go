package core

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
)

type Service struct {
	log         *slog.Logger
	db          DB
	xkcd        XKCD
	words       Words
	concurrency int

	running atomic.Bool
}

func NewService(
	log *slog.Logger,
	db DB,
	xkcd XKCD,
	words Words,
	concurrency int,
) (*Service, error) {
	if concurrency < 1 {
		return nil, fmt.Errorf("wrong concurrency specified: %d", concurrency)
	}
	return &Service{
		log:         log,
		db:          db,
		xkcd:        xkcd,
		words:       words,
		concurrency: concurrency,
	}, nil
}

func (s *Service) lockRun() error {
	if !s.running.CompareAndSwap(false, true) {
		return ErrAlreadyExists
	}
	return nil
}

func (s *Service) unlockRun() {
	s.running.Store(false)
}

func (s *Service) worker(ctx context.Context, jobs <-chan int) {
	for id := range jobs {
		info, err := s.xkcd.Get(ctx, id)
		if err != nil {
			s.log.Error("xkcd get failed", "id", id, "err", err)
			continue
		}

		// отдаем на нормализацию заголовок и описание
		norm, err := s.words.Norm(ctx, info.Title+" "+info.Description)
		if err != nil {
			s.log.Error("words norm failed", "id", id, "err", err)
			continue
		}

		c := Comics{
			ID:          info.ID,
			URL:         info.URL,
			Title:       info.Title,
			Description: info.Description,
			Words:       norm,
		}

		if err = s.db.Add(ctx, c); err != nil {
			s.log.Error("db add failed", "id", id, "err", err)
		}
	}
}

func (s *Service) Update(ctx context.Context) (err error) {
	if err := s.lockRun(); err != nil {
		return err
	}
	defer s.unlockRun()

	// последний номер id на xkcd
	last, err := s.xkcd.LastID(ctx)
	if err != nil {
		return err
	}

	// какие у нас уже есть в бд
	have, err := s.db.IDs(ctx)
	if err != nil {
		return err
	}

	// множество уже имеющихся id
	haveSet := make(map[int]bool, len(have))
	for _, id := range have {
		haveSet[id] = true
	}

	// список недостающих id
	missing := make([]int, 0, last)
	for id := 1; id <= last; id++ {
		if id == 404 {
			// 404-й комикс отсутствует на xkcd
			continue
		}
		if !haveSet[id] {
			missing = append(missing, id)
		}
	}

	if len(missing) == 0 {
		s.log.Info("no new comics to fetch")
		return nil
	}

	jobs := make(chan int, s.concurrency*2)
	var wg sync.WaitGroup

	for i := 0; i < s.concurrency; i++ {
		wg.Go(func() {
			s.worker(ctx, jobs)
		})
	}

	for _, id := range missing {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return ctx.Err()
		case jobs <- id:
		}
	}
	close(jobs)
	wg.Wait()

	return nil
}

func (s *Service) Stats(ctx context.Context) (ServiceStats, error) {
	dbStat, err := s.db.Stats(ctx)
	if err != nil {
		return ServiceStats{}, err
	}

	last, err := s.xkcd.LastID(ctx)
	if err != nil {
		return ServiceStats{}, err
	}

	holes := 0
	if last >= 404 {
		holes++
	}

	return ServiceStats{
		DBStats:     dbStat,
		ComicsTotal: last - holes,
	}, nil
}

func (s *Service) Status(ctx context.Context) ServiceStatus {
	if s.running.Load() {
		return StatusRunning
	}
	return StatusIdle
}

func (s *Service) Drop(ctx context.Context) error {
	return s.db.Drop(ctx)
}

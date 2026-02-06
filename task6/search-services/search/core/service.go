package core

import (
	"cmp"
	"context"
	"log/slog"
	"slices"
	"sync"
)

type Service struct {
	log   *slog.Logger
	db    DB
	words Words

	mu     sync.RWMutex
	index  map[string][]int
	comics map[int]Comic
}

func NewService(log *slog.Logger, db DB, words Words) *Service {
	return &Service{
		log:    log,
		db:     db,
		words:  words,
		index:  make(map[string][]int),
		comics: make(map[int]Comic),
	}
}

func (s *Service) Search(ctx context.Context, phrase string, limit int) ([]Comic, error) {

	if phrase == "" || limit <= 0 {
		return nil, ErrBadArguments
	}

	qwords, err := s.words.Norm(ctx, phrase)
	if err != nil {
		return nil, err
	}
	if len(qwords) == 0 {
		return nil, nil
	}

	qset := make(map[string]bool, len(qwords))
	for _, w := range qwords {
		qset[w] = true
	}

	comics, err := s.db.Search(ctx)
	if err != nil {
		return nil, err
	}

	type scored struct {
		comic   Comic
		matches int
		ratio   float64
	}

	scoredList := make([]scored, 0, len(comics))

	for _, c := range comics {
		if len(c.Words) == 0 {
			continue
		}
		matches := 0
		for _, w := range c.Words {
			if qset[w] {
				matches++
			}
		}
		if matches == 0 {
			continue
		}

		ratio := float64(matches) / float64(len(c.Words))
		scoredList = append(scoredList, scored{
			// у каждого комикса будет количество совпадений и кэф от общего текста
			comic:   c,
			matches: matches,
			ratio:   ratio,
		})
	}

	slices.SortFunc(scoredList, func(a, b scored) int {
		if a.matches != b.matches {
			return cmp.Compare(b.matches, a.matches) // по убыванию
		}
		if a.ratio != b.ratio {
			return cmp.Compare(b.ratio, a.ratio) // по убыванию
		}
		// Если matches и ratio одинаковы, то с меньшим id будет выше
		return cmp.Compare(a.comic.ID, b.comic.ID)

	})

	if limit > len(scoredList) {
		limit = len(scoredList)
	}

	res := make([]Comic, 0, limit)
	for i := 0; i < limit; i++ {
		res = append(res, scoredList[i].comic)
	}
	return res, nil
}

func (s *Service) RebuildIndex(ctx context.Context) error {

	comics, err := s.db.Search(ctx)
	if err != nil {
		return err
	}

	newIndex := make(map[string][]int)
	newComics := make(map[int]Comic, len(comics))

	for _, comic := range comics {
		newComics[comic.ID] = comic
		for _, w := range comic.Words {
			newIndex[w] = append(newIndex[w], comic.ID)
		}
	}
	// пока выполняем, никто не может читать
	s.mu.Lock()
	s.index = newIndex
	s.comics = newComics
	s.mu.Unlock()

	s.log.Info("search index rebuilt",
		"comics", len(newComics),
		"words", len(newIndex),
	)

	return nil
}

func (s *Service) IndexSearch(ctx context.Context, phrase string, limit int) ([]Comic, error) {

	if phrase == "" || limit <= 0 {
		return nil, ErrBadArguments
	}

	qwords, err := s.words.Norm(ctx, phrase)
	if err != nil {
		return nil, err
	}
	if len(qwords) == 0 {
		return nil, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.index) == 0 || len(s.comics) == 0 {
		return nil, nil
	}

	type scored struct {
		comic   Comic
		matches int
		ratio   float64
	}

	byId := make(map[int]*scored)

	for _, qword := range qwords {
		ids := s.index[qword]

		for _, id := range ids {
			sc, ok := byId[id]

			if !ok {
				c, exists := s.comics[id]
				if !exists {
					continue
				}
				sc = &scored{comic: c}
				byId[id] = sc
			}
			sc.matches++
		}
	}

	if len(byId) == 0 {
		return nil, nil
	}

	scoredList := make([]scored, 0, len(byId))
	for _, sc := range byId {
		wordCount := len(sc.comic.Words)
		if wordCount == 0 {
			continue
		}

		sc.ratio = float64(sc.matches) / float64(wordCount)
		scoredList = append(scoredList, *sc)
	}

	if len(scoredList) == 0 {
		return nil, nil
	}

	slices.SortFunc(scoredList, func(a, b scored) int {
		if a.matches != b.matches {
			return cmp.Compare(b.matches, a.matches) // по убыванию
		}
		if a.ratio != b.ratio {
			return cmp.Compare(b.ratio, a.ratio) // по убыванию
		}
		// Если matches и ratio одинаковы, то с меньшим id будет выше
		return cmp.Compare(a.comic.ID, b.comic.ID)
	})

	if limit > len(scoredList) {
		limit = len(scoredList)
	}

	res := make([]Comic, 0, limit)
	for i := 0; i < limit; i++ {
		res = append(res, scoredList[i].comic)
	}
	return res, nil
}

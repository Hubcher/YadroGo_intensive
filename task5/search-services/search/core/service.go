package core

import (
	"cmp"
	"context"
	"log/slog"
	"slices"
)

type Service struct {
	log   *slog.Logger
	db    DB
	words Words
}

func NewService(log *slog.Logger, db DB, words Words) *Service {
	return &Service{
		log:   log,
		db:    db,
		words: words,
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

	// любимые компараторы
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

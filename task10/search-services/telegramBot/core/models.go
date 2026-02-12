package core

type SearchComic struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

type SearchResponse struct {
	Comics []SearchComic `json:"comics"`
	Total  int           `json:"total"`
}

type UpdateStatsResponse struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

type UpdateStatusResponse struct {
	Status string `json:"status"`
}

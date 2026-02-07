package rest

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"yadro.com/course/api/core"
)

type PingResponse struct {
	Replies map[string]string `json:"replies"`
}

func NewPingHandler(log *slog.Logger, pingers map[string]core.Pinger) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		reply := PingResponse{
			Replies: make(map[string]string),
		}
		for name, pinger := range pingers {
			if err := pinger.Ping(r.Context()); err != nil {
				reply.Replies[name] = "unavailable"
				log.Error("one ot services is not available", "service", name)
				continue
			}
			reply.Replies[name] = "ok"
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(reply); err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}

type Authenticator interface {
	Login(user, password string) (string, error)
}

type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func NewLoginHandler(log *slog.Logger, auth Authenticator) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Error("cannot close request body", "error", err)
			}
		}()

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("cannot decode login request", "error", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		token, err := auth.Login(req.Name, req.Password)
		if err != nil {
			log.Error("cannot login", "error", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte(token)); err != nil {
			log.Error("cannot write login response", "error", err)
		}
	}
}

func NewUpdateHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		err := updater.Update(r.Context())
		if err != nil {
			log.Error("error while update", "error", err)
			if errors.Is(err, core.ErrAlreadyExists) {
				w.WriteHeader(http.StatusAccepted)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

type UpdateStatsResponse struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

func NewUpdateStatsHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		st, err := updater.Stats(r.Context())
		if err != nil {
			log.Error("error while stats", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(UpdateStatsResponse{
			WordsTotal:    st.WordsTotal,
			WordsUnique:   st.WordsUnique,
			ComicsFetched: st.ComicsFetched,
			ComicsTotal:   st.ComicsTotal,
		})
	}
}

type UpdateStatusResponse struct {
	Status string `json:"status"`
}

func NewUpdateStatusHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		st, err := updater.Status(r.Context())
		if err != nil {
			log.Error("error while status", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var status string
		switch st {
		case core.StatusUpdateIdle:
			status = "idle"
		case core.StatusUpdateRunning:
			status = "running"
		default:
			status = "unknown"
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(UpdateStatusResponse{Status: status})
		if err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}

func NewDropHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := updater.Drop(r.Context())
		if err != nil {
			log.Error("error while drop", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

type SearchComic struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

type SearchResponse struct {
	Comics []SearchComic `json:"comics"`
	Total  int           `json:"total"`
}

func NewSearchHandler(log *slog.Logger, search core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			log.Error("missing phrase")
			http.Error(w, "empty phrase", http.StatusBadRequest)
			return
		}

		const defaultLimit = 10
		limit := defaultLimit

		if l := r.URL.Query().Get("limit"); l != "" {
			val, err := strconv.Atoi(l)
			if err != nil || val <= 0 {
				http.Error(w, "invalid limit", http.StatusBadRequest)
				return
			}
			limit = val
		}

		comics, err := search.Search(r.Context(), phrase, limit)
		if err != nil {
			if errors.Is(err, core.ErrBadArguments) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			log.Error("search failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		reply := SearchResponse{
			Comics: make([]SearchComic, 0, len(comics)),
			Total:  len(comics),
		}

		for _, cmt := range comics {
			reply.Comics = append(reply.Comics, SearchComic{
				ID:  cmt.ID,
				URL: cmt.URL,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(reply); err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}

func NewIndexSearchHandler(log *slog.Logger, search core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			log.Error("missing phrase")
			http.Error(w, "empty phrase", http.StatusBadRequest)
			return
		}

		const defaultLimit = 10
		limit := defaultLimit

		if l := r.URL.Query().Get("limit"); l != "" {
			val, err := strconv.Atoi(l)
			if err != nil || val <= 0 {
				http.Error(w, "invalid limit", http.StatusBadRequest)
				return
			}
			limit = val
		}

		comics, err := search.IndexSearch(r.Context(), phrase, limit)
		if err != nil {
			if errors.Is(err, core.ErrBadArguments) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			log.Error("index search failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		reply := SearchResponse{
			Comics: make([]SearchComic, 0, len(comics)),
			Total:  len(comics),
		}

		for _, cmt := range comics {
			reply.Comics = append(reply.Comics, SearchComic{
				ID:  cmt.ID,
				URL: cmt.URL,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(reply); err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}

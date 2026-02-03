package rest

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"yadro.com/course/api/core"
)

type wordsHandler struct {
	log        *slog.Logger
	normalizer core.Normalizer
}

func NewWordsHandler(log *slog.Logger, n core.Normalizer) http.Handler {
	return &wordsHandler{log: log, normalizer: n}
}

func (h *wordsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	phrase := r.URL.Query().Get("phrase")

	if phrase == "" {
		http.Error(w, "missing phrase", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	words, err := h.normalizer.Norm(ctx, phrase)
	if err != nil {
		if errors.Is(err, core.ErrBadArguments) {
			http.Error(w, "bad phrase", http.StatusBadRequest)
			return
		}
		h.log.Error("words norm failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	type resp struct {
		Words []string `json:"words"`
		Total int      `json:"total"`
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp{Words: words, Total: len(words)})
}

func NewPingHandler(log *slog.Logger, pingers map[string]core.Pinger) http.HandlerFunc {

	type pingResponse struct {
		Replies map[string]string `json:"replies"`
	}

	return func(w http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		replies := make(map[string]string, len(pingers))
		for name, p := range pingers {
			if err := p.Ping(ctx); err != nil {
				replies[name] = "unavailable"
			} else {
				replies[name] = "ok"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(pingResponse{Replies: replies})
	}
}

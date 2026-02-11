package middleware

import (
	"net/http"
)

func Concurrency(next http.HandlerFunc, limit int) http.HandlerFunc {
	if limit <= 0 {
		return next
	}

	sema := make(chan struct{}, limit)

	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case sema <- struct{}{}:
			defer func() { <-sema }()
			next(w, r)
		default:
			http.Error(w, "too many concurrent requests", http.StatusServiceUnavailable)
		}
	}
}

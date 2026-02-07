package middleware

import (
	"net/http"
	"strings"
)

const (
	authorizationHeader = "Authorization"
	prefix              = "Token "
)

type TokenVerifier interface {
	Verify(token string) error
}

func Auth(next http.HandlerFunc, verifier TokenVerifier) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get(authorizationHeader)
		// достаем строку типа "Authorization: Token abcABC6.."

		// отсекаем невалидные "" и по сути другие форматы: OAuth, где что-то типа Authorization: Bearer something
		if !strings.HasPrefix(authHeader, prefix) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimSpace(authHeader[len(prefix):]) // берём токен
		if tokenString == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if err := verifier.Verify(tokenString); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

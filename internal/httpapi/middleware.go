package httpapi

import (
	"log"
	"net/http"
)

func contentMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("HTTP panic: %v", recovered)
				handleError(w, requestError{"internal_error", "服务内部错误"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

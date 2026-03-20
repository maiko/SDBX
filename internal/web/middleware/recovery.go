package middleware

import (
	"log"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery middleware recovers from panics and returns 500 error
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Use a logger that writes through log.Writer() so output
				// destination follows log.SetOutput (important for testing).
				logger := slog.New(slog.NewTextHandler(log.Writer(), nil))
				logger.Error("Panic recovered",
					"error", err,
					"stack", string(debug.Stack()),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

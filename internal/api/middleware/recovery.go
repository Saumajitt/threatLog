package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/Saumajitt/threatLog/internal/model"
)

// Recovery recovers from panics and returns 500
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Error().
					Interface("error", err).
					Str("path", r.URL.Path).
					Msg("Panic recovered")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				
				json.NewEncoder(w).Encode(model.ErrorResponse{
					Error:   "internal_server_error",
					Message: "An unexpected error occurred",
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}
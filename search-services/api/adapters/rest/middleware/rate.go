package middleware

import (
	"net/http"

	"golang.org/x/time/rate"
)

func Rate(next http.HandlerFunc, rps int) http.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(rps), 1)
	return func(w http.ResponseWriter, r *http.Request) {
		err := limiter.Wait(r.Context())
		if err != nil {
			http.Error(w, "Cancel or timeout", http.StatusGatewayTimeout)
			return
		}
		next(w, r)
	}
}

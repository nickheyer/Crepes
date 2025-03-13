package middleware

import (
	"log"
	"net/http"
	"time"
)

// LOGGING MIDDLEWARE TO LOG HTTP REQUESTS
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// CALL THE NEXT HANDLER
		next.ServeHTTP(w, r)
		// LOG REQUEST DETAILS
		log.Printf("%s %s %s\n", r.Method, r.RequestURI, time.Since(start))
	})
}

// CORS MIDDLEWARE TO HANDLE CROSS-ORIGIN REQUESTS
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// SET CORS HEADERS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// HANDLE PREFLIGHT REQUESTS
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// CALL THE NEXT HANDLER
		next.ServeHTTP(w, r)
	})
}

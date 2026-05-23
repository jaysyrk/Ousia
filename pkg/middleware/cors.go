package middleware

import "net/http"

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         string
}

func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-Request-Id", "X-Trace-Id"},
		MaxAge:         "86400",
	}
}

func CORS(cfg CORSConfig) Middleware {
	allowed := make(map[string]bool)
	for _, o := range cfg.AllowedOrigins {
		allowed[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if allowed["*"] || allowed[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", join(cfg.AllowedMethods))
			w.Header().Set("Access-Control-Allow-Headers", join(cfg.AllowedHeaders))
			w.Header().Set("Access-Control-Max-Age", cfg.MaxAge)

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func join(s []string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += ", "
		}
		result += v
	}
	return result
}

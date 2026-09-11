package middleware

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimit returns middleware that limits requests per IP using a token bucket algorithm.
// rps is the sustained requests per second; burst is the maximum burst size.
func RateLimit(rps float64, burst int) func(http.Handler) http.Handler {
	var mu sync.Mutex
	limiters := make(map[string]*rateLimiterEntry)

	// Cleanup stale entries every minute
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			mu.Lock()
			for ip, entry := range limiters {
				if time.Since(entry.lastSeen) > 5*time.Minute {
					delete(limiters, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)

			mu.Lock()
			entry, exists := limiters[ip]
			if !exists {
				entry = &rateLimiterEntry{
					limiter: rate.NewLimiter(rate.Limit(rps), burst),
				}
				limiters[ip] = entry
			}
			entry.lastSeen = time.Now()
			mu.Unlock()

			if !entry.limiter.Allow() {
				w.Header().Set("Retry-After", strconv.Itoa(int(1/rps)+1))
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func extractIP(r *http.Request) string {
	// Check X-Forwarded-For first (from reverse proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (client IP)
		if i := strings.IndexByte(xff, ','); i != -1 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

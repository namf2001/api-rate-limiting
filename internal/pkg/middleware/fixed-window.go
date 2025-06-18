package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type FixedWindow struct {
	count           int
	reset           time.Time
	lastRequestTime time.Time
}

func ResetFixedWindows() {
	mu.Lock()
	defer mu.Unlock()

	for ip, window := range fixedWindows {
		if time.Since(window.lastRequestTime) > time.Minute {
			delete(fixedWindows, ip)
		}
	}
}

func CheckFixedWindowLimit(ip string, limit int, window time.Duration) bool {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	client, exists := fixedWindows[ip]

	if !exists || now.After(client.reset) {
		fixedWindows[ip] = &FixedWindow{
			count:           1,
			reset:           now.Add(window),
			lastRequestTime: now,
		}
		return true
	}

	client.lastRequestTime = now

	if client.count >= limit {
		return false
	}

	client.count++
	return true
}

// FixedWindowMiddleware implements a fixed window rate limiting algorithm.
func FixedWindowMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		clientIP := GetClientIP(ctx)

		if !CheckFixedWindowLimit(clientIP, limit, window) {
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too many requests, please try again later.",
				"message": "You have exceeded the rate limit. Please wait before making more requests.",
			})
			return
		}

		ctx.Next()
	}
}

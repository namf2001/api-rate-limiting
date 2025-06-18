package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type SlidingWindow struct {
	requests        []time.Time
	lastRequestTime time.Time
}

func ResetSlidingWindows() {
	mu.Lock()
	defer mu.Unlock()

	for ip, window := range slidingWindows {
		if time.Since(window.lastRequestTime) > time.Minute {
			delete(slidingWindows, ip)
		}
	}
}

func cleanOldRequests(requests []time.Time, cutoff time.Time) []time.Time {
	validRequests := make([]time.Time, 0)
	for _, req := range requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	return validRequests
}

func CheckSlidingWindowLimit(ip string, limit int, window time.Duration) bool {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	client, exists := slidingWindows[ip]
	if !exists {
		slidingWindows[ip] = &SlidingWindow{
			requests:        []time.Time{now},
			lastRequestTime: now,
		}
		return true
	}

	client.lastRequestTime = now
	client.requests = cleanOldRequests(client.requests, cutoff)

	if len(client.requests) >= limit {
		return false
	}

	client.requests = append(client.requests, now)
	return true
}

// SlidingWindowMiddleware implements a sliding window rate limiting algorithm.
func SlidingWindowMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		clientIP := GetClientIP(ctx)

		if !CheckSlidingWindowLimit(clientIP, limit, window) {
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too many requests, please try again later.",
				"message": "You have exceeded the rate limit. Please wait before making more requests.",
			})
			return
		}

		ctx.Next()
	}
}

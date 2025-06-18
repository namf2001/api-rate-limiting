package middleware

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type TokenBucket struct {
	limiter         *rate.Limiter
	lastRequestTime time.Time
}

func ResetTokenBuckets(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mu.Lock()
			for ip, bucket := range tokenBuckets {
				if time.Since(bucket.lastRequestTime) > time.Minute {
					delete(tokenBuckets, ip)
				}
			}
			mu.Unlock()
		}
	}
}

func GetClientIP(ctx *gin.Context) string {
	clientIP := ctx.ClientIP()
	if clientIP == "" {
		// Parse IP from RemoteAddr to exclude port number
		host, _, err := net.SplitHostPort(ctx.Request.RemoteAddr)
		if err != nil {
			// If SplitHostPort fails, use RemoteAddr as-is (fallback)
			clientIP = ctx.Request.RemoteAddr
		} else {
			clientIP = host
		}
	}
	return clientIP
}

func RateLimit(ip string, rateLimit, burst int) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	if bucket, exists := tokenBuckets[ip]; exists {
		// Only update the last request time, no limiter reset logic
		bucket.lastRequestTime = time.Now()
		return bucket.limiter
	}

	// Create new bucket if it doesn't exist
	newBucket := &TokenBucket{
		limiter:         rate.NewLimiter(rate.Limit(rateLimit), burst),
		lastRequestTime: time.Now(),
	}
	tokenBuckets[ip] = newBucket
	return newBucket.limiter
}

// TokenBucketMiddleware implements a token bucket rate limiting algorithm.
func TokenBucketMiddleware(rateLimit, burst int) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		clientIP := GetClientIP(ctx)
		limiter := RateLimit(clientIP, rateLimit, burst)
		if !limiter.Allow() {
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too many requests, please try again later.",
				"message": "You have exceeded the rate limit. Please wait before making more requests.",
			})
			return
		}
		ctx.Next()
	}
}

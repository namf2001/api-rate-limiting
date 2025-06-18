package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type TokenBucket struct {
	limiter         *rate.Limiter
	lastRequestTime time.Time
}

func ResetTokenBuckets() {
	mu.Lock()
	defer mu.Unlock()

	for ip, bucket := range tokenBuckets {
		if time.Since(bucket.lastRequestTime) > time.Minute {
			delete(tokenBuckets, ip)
		}
	}
}

func GetClientIP(ctx *gin.Context) string {
	clientIP := ctx.ClientIP()
	if clientIP == "" {
		clientIP = ctx.Request.RemoteAddr
	}
	return clientIP
}

func RateLimit(ip string, rateLimit, burst int) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	if bucket, exists := tokenBuckets[ip]; exists {
		if time.Since(bucket.lastRequestTime) > time.Minute {
			bucket.limiter = rate.NewLimiter(rate.Limit(rateLimit), burst)
		}
		bucket.lastRequestTime = time.Now()
		return bucket.limiter
	}

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

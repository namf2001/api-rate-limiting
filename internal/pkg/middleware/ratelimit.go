package middleware

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	limiter         *rate.Limiter
	lastRequestTime time.Time
}

var (
	mu      sync.Mutex
	clients = make(map[string]*Client)
)

func GetClientIP(ctx *gin.Context) string {
	clientIP := ctx.ClientIP()
	if clientIP == "" {
		clientIP = ctx.Request.RemoteAddr
	}
	return clientIP
}

func RateLimit(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	if client, exists := clients[ip]; exists {
		if time.Since(client.lastRequestTime) > time.Minute {
			client.limiter = rate.NewLimiter(1, 3)
		}
		client.lastRequestTime = time.Now()
		//log.Printf("Client[%s] reused – limiter=%+v, lastRequest=%s", ip, client.limiter, client.lastRequestTime.Format(time.TimeOnly))
		return client.limiter
	}

	newClient := &Client{
		limiter:         rate.NewLimiter(1, 3),
		lastRequestTime: time.Now(),
	}
	clients[ip] = newClient
	//log.Printf("Client[%s] created – limiter=%+v, lastRequest=%s", ip, newClient.limiter, newClient.lastRequestTime.Format(time.TimeOnly))
	return newClient.limiter
}

func CleanupClients() {
	mu.Lock()
	defer mu.Unlock()

	for ip, client := range clients {
		if time.Since(client.lastRequestTime) > time.Minute {
			delete(clients, ip)
		}
	}
}

// RateLimitMiddleware ab -c 2 -n 30 http://localhost:8080/
func RateLimitMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		clientIP := GetClientIP(ctx)
		limiter := RateLimit(clientIP)
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

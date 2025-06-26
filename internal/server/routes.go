package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"api-rate-limiting/internal/pkg/middleware"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	// Create context with cancellation for graceful shutdown of background goroutines
	ctx, cancel := context.WithCancel(context.Background())

	// Store the cancel function for later use during server shutdown
	s.cancel = cancel

	go middleware.ResetTokenBuckets(ctx)
	go middleware.ResetFixedWindows(ctx)
	go middleware.ResetSlidingWindows(ctx)

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"}, // Add your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Enable cookies/auth
	}))

	r.GET("/", s.HelloWorldHandler)

	r.GET("/health", s.healthHandler)

	// Instagram download endpoint
	r.POST("/instagram/download", s.InstagramDownloadHandler)

	// Fixed Window: 3 request/10 seconds
	r.GET("/fixed", middleware.FixedWindowMiddleware(3, time.Second), s.TestHandler("Fixed Window"))

	// Sliding Window: 5 request/30 seconds
	r.GET("/sliding", middleware.SlidingWindowMiddleware(5, 30*time.Second), s.TestHandler("Sliding Window"))

	// Token Bucket: 1 token/second with a burst of 3 tokens
	r.GET("/token-bucket", middleware.TokenBucketMiddleware(1, 3), s.TestHandler("Token Bucket"))
	return r
}

func (s *Server) HelloWorldHandler(c *gin.Context) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	c.JSON(http.StatusOK, resp)
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.db.Health())
}

func (s *Server) TestHandler(algorithm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"algorithm": algorithm,
			"message":   "Request successful",
			"time":      time.Now().Format(time.TimeOnly),
		})
	}
}

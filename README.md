# API-RATE-LIMITING

High-Performance Rate Limiting System for Go APIs

<div align="center">
  <img src="https://img.shields.io/badge/last%20commit-today-blue" alt="last commit" />
  <img src="https://img.shields.io/badge/go-100%25-blue" alt="go" />
  <img src="https://img.shields.io/badge/algorithms-3-blue" alt="algorithms" />
</div>

<div align="center">
  <h3><a href="http://localhost:8080" target="_blank">🚀 Local Demo: http://localhost:8080</a></h3>
</div>

# API Rate Limiting System

A high-performance rate limiting system built with Go that provides multiple rate limiting algorithms to control API request rates.

## Features

### Instagram Media Downloader

The API provides an endpoint to download media (images and videos) from Instagram:

- **Extract downloadable URLs**: Convert Instagram post/reel URLs to direct media URLs
- **Support for images and videos**: Automatically detects and handles both media types
- **Error handling**: Proper handling for private accounts, non-existent media, and invalid URLs

### Rate Limiting Algorithms

1. **Fixed Window Rate Limiting**
   - Divides time into fixed windows
   - Simple and memory efficient
   - Modular architecture with separate functions

2. **Sliding Window Rate Limiting**
   - More flexible than fixed window
   - Provides smoother request distribution
   - Enhanced with helper functions for cleanup

3. **Token Bucket Algorithm**
   - Dynamic request rate adjustment
   - Allows burst traffic within limits
   - Fully modular design with token management

### Architecture

Modular architecture with separated concerns:
- **Check Functions**: Core rate limiting logic
- **Reset Functions**: Automatic cleanup of inactive clients
- **Helper Functions**: Utility functions for common operations

## Code Structure

```
internal/pkg/middleware/
├── common.go           # Shared utilities
├── fixed-window.go     # Fixed window algorithm
├── sliding-window.go   # Sliding window algorithm
└── token-bucket.go     # Token bucket algorithm
```

## Getting Started

### Commands

```bash
# Build and run
make build
make run

# Development
make watch      # Live reload
make test       # Run tests

# Docker
make docker-run   # Start containers
make docker-down  # Stop containers
```

## Usage

### As Middleware

```go
import "github.com/gin-gonic/gin"

router := gin.Default()

// Apply rate limiting
router.Use(middleware.FixedWindowMiddleware(100, time.Hour))
router.Use(middleware.SlidingWindowMiddleware(100, time.Hour))
router.Use(middleware.TokenBucketMiddleware(100, time.Second))
```

### Individual Functions

```go
// Check rate limits
allowed := middleware.CheckFixedWindowLimit("192.168.1.1", 100, time.Hour)
allowed := middleware.CheckSlidingWindowLimit("192.168.1.1", 100, time.Hour)
allowed := middleware.CheckTokenBucketLimit("192.168.1.1", 100, time.Second)

// Cleanup inactive clients
middleware.ResetFixedWindows()
middleware.ResetSlidingWindows()
middleware.ResetTokenBuckets()
```

### Instagram Downloader API

Use the Instagram downloader endpoint to extract direct media URLs:

```bash
# Download an Instagram image or video
curl -X POST http://localhost:8080/instagram/download \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.instagram.com/p/EXAMPLE_POST_ID/"}'
```

Example response:

```json
{
  "download_url": "https://scontent.cdninstagram.com/v/t51.2885-15/123456789_123456789_123456789_n.jpg?...",
  "media_type": "image"
}
```

Error responses:

For invalid URL:
```json
{
  "error": "Invalid Instagram URL"
}
```

For private account:
```json
{
  "error": "media is from a private account"
}
```

For not found media:
```json
{
  "error": "media not found or has been deleted"
}
```

## Testing

### Unit Tests
```bash
make test       # Unit tests
make itest      # Integration tests
```

### Load Testing

Use Apache Bench (ab) to test rate limiting performance:

```bash
# Install Apache Bench (if not already installed)
# macOS: brew install httpie
# Ubuntu: sudo apt-get install apache2-utils

# Test Fixed Window (assuming endpoint exists)
ab -c 2 -n 30 http://localhost:8080/fixed

# Test Sliding Window
ab -c 2 -n 30 http://localhost:8080/sliding

# Test Token Bucket
ab -c 2 -n 30 http://localhost:8080/token

# More intensive testing
ab -c 10 -n 100 http://localhost:8080/sliding
ab -c 20 -n 200 http://localhost:8080/fixed
```

### Other Load Testing Tools

```bash
# Using curl for simple testing
for i in {1..10}; do curl http://localhost:8080/sliding; done

# Using wrk (install: brew install wrk)
wrk -t4 -c10 -d30s http://localhost:8080/sliding

# Using hey (install: go install github.com/rakyll/hey@latest)
hey -n 100 -c 10 http://localhost:8080/sliding
```

### Expected Test Results

When testing rate limits, you should see:
- **200 OK** responses until limit is reached
- **429 Too Many Requests** when limit exceeded
- Different behavior patterns for each algorithm:
  - Fixed Window: Sharp cutoff at window boundary
  - Sliding Window: Gradual enforcement
  - Token Bucket: Burst allowance then steady rate

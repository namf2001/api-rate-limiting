package middleware

import "sync"

var (
	mu             sync.Mutex
	fixedWindows   = make(map[string]*FixedWindow)
	slidingWindows = make(map[string]*SlidingWindow)
	tokenBuckets   = make(map[string]*TokenBucket)
)

package apiclient

import (
	"math/rand/v2"
	"time"
)

type retryConfig struct {
	Max  int           // total attempts incl. first
	Base time.Duration // first backoff
}

var defaultRetry = retryConfig{Max: 3, Base: 200 * time.Millisecond}

func (rc retryConfig) sleepFor(attempt int) time.Duration {
	d := rc.Base << attempt
	jitter := time.Duration(rand.Int64N(int64(rc.Base)))
	return d + jitter
}

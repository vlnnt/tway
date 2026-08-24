package client

import (
	"math/rand"
	"time"
)

const (
	retryBaseDelay = time.Second
	retryMaxDelay  = 30 * time.Second
	retryMaxJitter = 500 * time.Millisecond
)

func RetryDelay(
	attempt int,
) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	delay := retryBaseDelay
	for i := 1; i < attempt; i++ {
		if delay >= retryMaxDelay/2 {
			delay = retryMaxDelay
			break
		}

		delay *= 2
	}

	jitter := time.Duration(rand.Int63n(int64(retryMaxJitter) + 1))
	delay += jitter

	if delay > retryMaxDelay {
		return retryMaxDelay
	}

	return delay
}

package backoff

import (
	"math"
	"time"
)

func ExponentialJitter(base, max time.Duration, attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	mul := math.Pow(2, float64(attempt-1))
	d := min(time.Duration(float64(base)*mul), max)

	// simple jitter: +/- 20%
	j := time.Duration(float64(d) * 0.2)
	return d - j + time.Duration(int64(j)*time.Now().UnixNano()%int64(2*j))
}

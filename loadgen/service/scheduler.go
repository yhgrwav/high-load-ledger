package service

import (
	"context"
	"hash/fnv"
	"math"
	"math/rand"
	"time"
)

func poissonDelay(rps float64) time.Duration {
	u := rand.Float64()
	if u <= 0 {
		u = 1e-12
	}
	seconds := -math.Log(u) / rps
	return time.Duration(seconds * float64(time.Second))
}

func streamSeed(stream string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(stream))
	return int64(h.Sum64())
}

func runPoissonStream(
	ctx context.Context,
	stream string,
	rps float64,
	jobs chan<- TransferJob,
	build func() (TransferJob, bool),
	metrics *Metrics,
	stats *LoadStats,
) {
	if rps <= 0 {
		return
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano() ^ streamSeed(stream)))
	nextAt := time.Now()

	for {
		wait := time.Until(nextAt)
		if wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}

		job, ok := build()
		if ok {
			job.Kind = stream
			select {
			case jobs <- job:
				metrics.RecordDispatched(stream)
				stats.RecordDispatched(stream)
			case <-ctx.Done():
				return
			}
		}

		nextAt = nextAt.Add(poissonDelay(rps))
		if nextAt.Before(time.Now()) {
			nextAt = time.Now()
		}
	}
}

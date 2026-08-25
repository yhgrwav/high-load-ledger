package service

import (
	"context"
	"hash/fnv"
	"math"
	"math/rand"
	"time"
)

// maxCatchUp limits how far a stream may burst after backpressure.
// Without any catch-up, a full job queue permanently collapses target RPS.
const maxCatchUp = 250 * time.Millisecond

func poissonDelay(rng *rand.Rand, rps float64) time.Duration {
	u := rng.Float64()
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
	drop func(TransferJob),
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
		if !ok {
			// Do not burn a rate token when we failed to build a job (empty pool / no funded pair).
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Millisecond):
			}
			continue
		}

		job.Kind = stream
		select {
		case jobs <- job:
			metrics.RecordDispatched(stream)
			stats.RecordDispatched(stream)
		case <-ctx.Done():
			if drop != nil {
				drop(job)
			}
			return
		}

		nextAt = nextAt.Add(poissonDelay(rng, rps))
		// Bound catch-up instead of hard-resetting to now (old reset permanently under-dispatched).
		if lag := time.Since(nextAt); lag > maxCatchUp {
			nextAt = time.Now().Add(-maxCatchUp)
		}
	}
}

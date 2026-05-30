package service

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	loadgenconfig "high-load-ledger/loadgen/config"
)

const achievedTolerance = 0.05

type LoadStats struct {
	dispatchedValid           atomic.Int64
	dispatchedInvalidBalance  atomic.Int64
	dispatchedInvalidCurrency atomic.Int64
	completedOK               atomic.Int64
	completedError            atomic.Int64

	lastDispatchedValid           atomic.Int64
	lastDispatchedInvalidBalance  atomic.Int64
	lastDispatchedInvalidCurrency atomic.Int64
	lastTick                      atomic.Int64
}

func (s *LoadStats) RecordDispatched(kind string) {
	switch kind {
	case StreamValid:
		s.dispatchedValid.Add(1)
	case StreamInvalidBalance:
		s.dispatchedInvalidBalance.Add(1)
	case StreamInvalidCurrency:
		s.dispatchedInvalidCurrency.Add(1)
	}
}

func (s *LoadStats) RecordCompleted(err error) {
	if err != nil {
		s.completedError.Add(1)
		return
	}
	s.completedOK.Add(1)
}

func (s *LoadStats) RunReporter(ctx context.Context, cfg *loadgenconfig.Config) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logFinal(cfg)
			return
		case <-ticker.C:
			rValid, rInvalid, rCurrency := s.rates()
			log.Printf(
				"loadgen: dispatch rate valid=%.1f/s (target %.0f) invalid_balance=%.1f/s (target %.0f) invalid_currency=%.1f/s (target %.0f) completed_ok=%d completed_error=%d",
				rValid, cfg.ValidRPS,
				rInvalid, cfg.InvalidRPS,
				rCurrency, cfg.InvalidCurrencyRPS,
				s.completedOK.Load(), s.completedError.Load(),
			)
		}
	}
}

func (s *LoadStats) rates() (valid, invalidBalance, invalidCurrency float64) {
	now := time.Now().UnixNano()
	prev := s.lastTick.Load()
	if prev == 0 {
		s.lastTick.Store(now)
		s.lastDispatchedValid.Store(s.dispatchedValid.Load())
		s.lastDispatchedInvalidBalance.Store(s.dispatchedInvalidBalance.Load())
		s.lastDispatchedInvalidCurrency.Store(s.dispatchedInvalidCurrency.Load())
		return 0, 0, 0
	}

	elapsed := float64(now-prev) / float64(time.Second)
	if elapsed <= 0 {
		return 0, 0, 0
	}

	curValid := s.dispatchedValid.Load()
	curInvalid := s.dispatchedInvalidBalance.Load()
	curCurrency := s.dispatchedInvalidCurrency.Load()

	valid = float64(curValid-s.lastDispatchedValid.Load()) / elapsed
	invalidBalance = float64(curInvalid-s.lastDispatchedInvalidBalance.Load()) / elapsed
	invalidCurrency = float64(curCurrency-s.lastDispatchedInvalidCurrency.Load()) / elapsed

	s.lastTick.Store(now)
	s.lastDispatchedValid.Store(curValid)
	s.lastDispatchedInvalidBalance.Store(curInvalid)
	s.lastDispatchedInvalidCurrency.Store(curCurrency)

	return valid, invalidBalance, invalidCurrency
}

func (s *LoadStats) logFinal(cfg *loadgenconfig.Config) {
	rValid, rInvalid, rCurrency := s.rates()
	log.Printf(
		"loadgen final: dispatch valid=%.1f/s invalid_balance=%.1f/s invalid_currency=%.1f/s completed_ok=%d completed_error=%d",
		rValid, rInvalid, rCurrency, s.completedOK.Load(), s.completedError.Load(),
	)

	checks := []struct {
		stream   string
		target   float64
		achieved float64
	}{
		{StreamValid, cfg.ValidRPS, rValid},
		{StreamInvalidBalance, cfg.InvalidRPS, rInvalid},
		{StreamInvalidCurrency, cfg.InvalidCurrencyRPS, rCurrency},
	}

	passed := true
	for _, c := range checks {
		if err := ValidateAchieved(c.stream, c.target, c.achieved, achievedTolerance); err != nil {
			log.Printf("loadgen WARN: %v", err)
			passed = false
		}
	}
	if passed {
		log.Println("loadgen: dispatch rates within target tolerance (±5%)")
	}
}

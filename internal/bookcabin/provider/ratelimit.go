package provider

import (
	"context"
	"sync"
	"time"

	"github.com/shandysiswandi/gobookcabin/internal/bookcabin/entity"
)

type rateLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	last     time.Time
}

func newRateLimiter(interval time.Duration) *rateLimiter {
	return &rateLimiter{interval: interval}
}

func (r *rateLimiter) Wait(ctx context.Context) error {
	if r.interval <= 0 {
		return nil
	}
	for {
		now := time.Now()
		r.mu.Lock()
		if r.last.IsZero() || now.Sub(r.last) >= r.interval {
			r.last = now
			r.mu.Unlock()
			return nil
		}
		wait := r.interval - now.Sub(r.last)
		r.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

type rateLimitedProvider struct {
	provider Provider
	limiter  *rateLimiter
}

func NewRateLimitedProvider(p Provider, interval time.Duration) Provider {
	return &rateLimitedProvider{
		provider: p,
		limiter:  newRateLimiter(interval),
	}
}

func (r *rateLimitedProvider) Name() string {
	return r.provider.Name()
}

func (r *rateLimitedProvider) Search(ctx context.Context, req SearchRequest) ([]entity.Flight, error) {
	if err := r.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	return r.provider.Search(ctx, req)
}

package chaos

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

// LatencyConfig configures the latency injector.
type LatencyConfig struct {
	MinDelay   time.Duration
	MaxDelay   time.Duration
	JitterPct  float64 // 0.0 to 1.0, percentage of jitter added to base delay
	Enabled    bool
}

// LatencyInjector adds configurable latency to operations.
type LatencyInjector struct {
	mu     sync.RWMutex
	config LatencyConfig
	stats  LatencyStats
}

// LatencyStats tracks latency injection statistics.
type LatencyStats struct {
	TotalCalls    int64
	DelaysApplied int64
	TotalDelay    time.Duration
	MaxObserved   time.Duration
}

// NewLatencyInjector creates a new latency injector.
func NewLatencyInjector(cfg LatencyConfig) *LatencyInjector {
	return &LatencyInjector{
		config: cfg,
	}
}

// UpdateConfig atomically updates the configuration.
func (l *LatencyInjector) UpdateConfig(cfg LatencyConfig) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config = cfg
}

// GetConfig returns the current configuration.
func (l *LatencyInjector) GetConfig() LatencyConfig {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config
}

// GetStats returns a copy of the current stats.
func (l *LatencyInjector) GetStats() LatencyStats {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.stats
}

// Delay sleeps for a random duration within the configured range.
// Returns the actual delay applied.
func (l *LatencyInjector) Delay(ctx context.Context) (time.Duration, error) {
	l.mu.Lock()
	cfg := l.config
	l.stats.TotalCalls++

	if !cfg.Enabled {
		l.mu.Unlock()
		return 0, nil
	}

	delay := l.randomDelay(cfg)
	l.stats.DelaysApplied++
	l.stats.TotalDelay += delay
	if delay > l.stats.MaxObserved {
		l.stats.MaxObserved = delay
	}
	l.mu.Unlock()

	select {
	case <-time.After(delay):
		return delay, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// Wrap adds latency before executing the given function.
func (l *LatencyInjector) Wrap(fn func() error) func() error {
	return func() error {
		if _, err := l.Delay(context.Background()); err != nil {
			return err
		}
		return fn()
	}
}

// WrapWithContext adds latency with context support.
func (l *LatencyInjector) WrapWithContext(ctx context.Context, fn func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if _, err := l.Delay(ctx); err != nil {
			return err
		}
		return fn(ctx)
	}
}

func (l *LatencyInjector) randomDelay(cfg LatencyConfig) time.Duration {
	if cfg.MinDelay >= cfg.MaxDelay {
		return cfg.MinDelay
	}

	base := cfg.MinDelay + time.Duration(rand.Int63n(int64(cfg.MaxDelay-cfg.MinDelay)))

	if cfg.JitterPct > 0 {
		jitter := time.Duration(float64(base) * cfg.JitterPct * (rand.Float64()*2 - 1))
		base += jitter
	}

	if base < 0 {
		base = 0
	}
	return base
}

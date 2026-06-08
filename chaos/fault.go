package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// FaultType describes the kind of fault to inject.
type FaultType string

const (
	FaultLatency  FaultType = "latency"
	FaultError    FaultType = "error"
	FaultTimeout  FaultType = "timeout"
	FaultPanic    FaultType = "panic"
	FaultSlowRead FaultType = "slow_read"
)

// FaultConfig configures a fault injection.
type FaultConfig struct {
	Type        FaultType
	Probability float64       // 0.0 to 1.0
	Delay       time.Duration // for latency faults
	ErrorMsg    string        // for error faults
	Enabled     bool
}

// FaultInjector injects faults into function calls.
type FaultInjector struct {
	mu     sync.RWMutex
	config FaultConfig
	stats  FaultStats
}

// FaultStats tracks injection statistics.
type FaultStats struct {
	TotalCalls   int64
	FaultsInjected int64
	ErrorsInjected int64
	DelaysInjected int64
}

// NewFaultInjector creates a new fault injector with the given config.
func NewFaultInjector(cfg FaultConfig) *FaultInjector {
	return &FaultInjector{
		config: cfg,
	}
}

// UpdateConfig atomically updates the fault configuration.
func (f *FaultInjector) UpdateConfig(cfg FaultConfig) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.config = cfg
}

// GetConfig returns the current fault configuration.
func (f *FaultInjector) GetConfig() FaultConfig {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.config
}

// GetStats returns a copy of the current stats.
func (f *FaultInjector) GetStats() FaultStats {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.stats
}

// ResetStats zeroes the statistics.
func (f *FaultInjector) ResetStats() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stats = FaultStats{}
}

// Inject evaluates whether to inject a fault and does so if triggered.
// Returns an error if a fault was injected, nil otherwise.
func (f *FaultInjector) Inject(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.stats.TotalCalls++

	if !f.config.Enabled {
		return nil
	}

	if rand.Float64() >= f.config.Probability {
		return nil
	}

	switch f.config.Type {
	case FaultLatency:
		f.stats.FaultsInjected++
		f.stats.DelaysInjected++
		delay := f.config.Delay
		if delay == 0 {
			delay = 100 * time.Millisecond
		}
		select {
		case <-time.After(delay):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}

	case FaultError:
		f.stats.FaultsInjected++
		f.stats.ErrorsInjected++
		msg := f.config.ErrorMsg
		if msg == "" {
			msg = "chaos fault injected"
		}
		return fmt.Errorf("fault: %s", msg)

	case FaultTimeout:
		f.stats.FaultsInjected++
		delay := f.config.Delay
		if delay == 0 {
			delay = 30 * time.Second
		}
		select {
		case <-time.After(delay):
			return fmt.Errorf("fault: operation timed out after %s", delay)
		case <-ctx.Done():
			return ctx.Err()
		}

	case FaultPanic:
		f.stats.FaultsInjected++
		panic("chaos fault: injected panic")

	default:
		return nil
	}
}

// Wrap wraps a function with fault injection.
func (f *FaultInjector) Wrap(fn func() error) func() error {
	return func() error {
		if err := f.Inject(context.Background()); err != nil {
			return err
		}
		return fn()
	}
}

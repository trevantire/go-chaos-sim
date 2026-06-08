package chaos

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PartitionConfig configures a network partition.
type PartitionConfig struct {
	// AffectedEndpoints lists endpoints (host:port) that should be partitioned.
	AffectedEndpoints []string

	// Mode determines the partition behavior.
	Mode PartitionMode

	// Duration is how long the partition lasts (0 = indefinite until stopped).
	Duration time.Duration

	// DropProbability is the chance [0.0, 1.0] that a request is dropped (for partial partitions).
	DropProbability float64
}

// PartitionMode describes how the partition behaves.
type PartitionMode string

const (
	// PartitionFull drops all traffic to affected endpoints.
	PartitionFull PartitionMode = "full"

	// PartitionPartial drops traffic randomly based on DropProbability.
	PartitionPartial PartitionMode = "partial"

	// PartitionBlackhole accepts connections but never responds.
	PartitionBlackhole PartitionMode = "blackhole"
)

// NetworkPartition simulates network partitions between services.
type NetworkPartition struct {
	mu       sync.RWMutex
	config   PartitionConfig
	active   bool
	cancelFn context.CancelFunc
	stats    PartitionStats
}

// PartitionStats tracks partition activity.
type PartitionStats struct {
	RequestsBlocked  int64
	RequestsAllowed  int64
	PartitionsActive int
}

// NewNetworkPartition creates a new network partition simulator.
func NewNetworkPartition() *NetworkPartition {
	return &NetworkPartition{}
}

// Activate starts the network partition with the given configuration.
func (np *NetworkPartition) Activate(cfg PartitionConfig) error {
	np.mu.Lock()
	defer np.mu.Unlock()

	if np.active {
		return fmt.Errorf("partition already active")
	}

	np.config = cfg
	np.active = true
	np.stats.PartitionsActive = len(cfg.AffectedEndpoints)

	if cfg.Duration > 0 {
		ctx, cancel := context.WithCancel(context.Background())
		np.cancelFn = cancel
		go func() {
			select {
			case <-time.After(cfg.Duration):
				np.Deactivate()
			case <-ctx.Done():
			}
		}()
	}

	return nil
}

// Deactivate stops the active partition.
func (np *NetworkPartition) Deactivate() {
	np.mu.Lock()
	defer np.mu.Unlock()

	if np.cancelFn != nil {
		np.cancelFn()
		np.cancelFn = nil
	}
	np.active = false
	np.stats.PartitionsActive = 0
}

// IsActive returns whether a partition is currently active.
func (np *NetworkPartition) IsActive() bool {
	np.mu.RLock()
	defer np.mu.RUnlock()
	return np.active
}

// GetStats returns a copy of the current stats.
func (np *NetworkPartition) GetStats() PartitionStats {
	np.mu.RLock()
	defer np.mu.RUnlock()
	return np.stats
}

// ShouldBlock checks if a request to the given endpoint should be blocked.
// Returns true if the request should be rejected, false if it should proceed.
func (np *NetworkPartition) ShouldBlock(endpoint string) bool {
	np.mu.Lock()
	defer np.mu.Unlock()

	if !np.active {
		return false
	}

	if !np.isAffected(endpoint) {
		return false
	}

	switch np.config.Mode {
	case PartitionFull:
		np.stats.RequestsBlocked++
		return true

	case PartitionPartial:
		if np.config.DropProbability > 0 && randomFloat() < np.config.DropProbability {
			np.stats.RequestsBlocked++
			return true
		}
		np.stats.RequestsAllowed++
		return false

	case PartitionBlackhole:
		np.stats.RequestsBlocked++
		return true

	default:
		return false
	}
}

// Wrap wraps a function that targets a specific endpoint.
func (np *NetworkPartition) Wrap(endpoint string, fn func() error) func() error {
	return func() error {
		if np.ShouldBlock(endpoint) {
			switch np.config.Mode {
			case PartitionBlackhole:
				// Simulate hanging connection
				time.Sleep(30 * time.Second)
				return fmt.Errorf("partition: connection to %s timed out (blackhole)", endpoint)
			default:
				return fmt.Errorf("partition: connection to %s refused", endpoint)
			}
		}
		return fn()
	}
}

func (np *NetworkPartition) isAffected(endpoint string) bool {
	if len(np.config.AffectedEndpoints) == 0 {
		return true // affect all if none specified
	}
	for _, ep := range np.config.AffectedEndpoints {
		if ep == endpoint {
			return true
		}
	}
	return false
}

func randomFloat() float64 {
	// Thread-safe random float in [0, 1)
	return float64(time.Now().UnixNano()%10000) / 10000.0
}

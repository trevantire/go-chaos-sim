package chaos

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/trevantire/go-chaos-sim/chaos"
)

// --- FaultInjector Tests ---

func TestFaultInjector_Disabled(t *testing.T) {
	fi := chaos.NewFaultInjector(chaos.FaultConfig{
		Enabled: false,
	})

	for i := 0; i < 100; i++ {
		if err := fi.Inject(context.Background()); err != nil {
			t.Fatalf("expected no error when disabled, got: %v", err)
		}
	}

	stats := fi.GetStats()
	if stats.FaultsInjected != 0 {
		t.Fatalf("expected 0 faults when disabled, got %d", stats.FaultsInjected)
	}
}

func TestFaultInjector_ErrorInjection(t *testing.T) {
	fi := chaos.NewFaultInjector(chaos.FaultConfig{
		Type:        chaos.FaultError,
		Probability: 1.0,
		Enabled:     true,
		ErrorMsg:    "test fault",
	})

	err := fi.Inject(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, err) {
		t.Fatalf("unexpected error message: %v", err)
	}

	stats := fi.GetStats()
	if stats.ErrorsInjected != 1 {
		t.Fatalf("expected 1 error injected, got %d", stats.ErrorsInjected)
	}
}

func TestFaultInjector_LatencyInjection(t *testing.T) {
	fi := chaos.NewFaultInjector(chaos.FaultConfig{
		Type:        chaos.FaultLatency,
		Probability: 1.0,
		Delay:       50 * time.Millisecond,
		Enabled:     true,
	})

	start := time.Now()
	err := fi.Inject(context.Background())
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected no error for latency, got: %v", err)
	}
	if elapsed < 40*time.Millisecond {
		t.Fatalf("expected delay >= 40ms, got %v", elapsed)
	}
}

func TestFaultInjector_ContextCancellation(t *testing.T) {
	fi := chaos.NewFaultInjector(chaos.FaultConfig{
		Type:        chaos.FaultLatency,
		Probability: 1.0,
		Delay:       5 * time.Second,
		Enabled:     true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := fi.Inject(ctx)
	if err == nil {
		t.Fatal("expected context error, got nil")
	}
}

func TestFaultInjector_Wrap(t *testing.T) {
	fi := chaos.NewFaultInjector(chaos.FaultConfig{
		Enabled: false,
	})

	called := false
	wrapped := fi.Wrap(func() error {
		called = true
		return nil
	})

	if err := wrapped(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected wrapped function to be called")
	}
}

func TestFaultInjector_UpdateConfig(t *testing.T) {
	fi := chaos.NewFaultInjector(chaos.FaultConfig{Enabled: false})

	fi.UpdateConfig(chaos.FaultConfig{
		Type:        chaos.FaultError,
		Probability: 1.0,
		Enabled:     true,
		ErrorMsg:    "updated",
	})

	cfg := fi.GetConfig()
	if !cfg.Enabled {
		t.Fatal("expected config to be enabled after update")
	}
	if cfg.ErrorMsg != "updated" {
		t.Fatalf("expected error msg 'updated', got '%s'", cfg.ErrorMsg)
	}
}

// --- LatencyInjector Tests ---

func TestLatencyInjector_Disabled(t *testing.T) {
	li := chaos.NewLatencyInjector(chaos.LatencyConfig{Enabled: false})

	delay, err := li.Delay(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if delay != 0 {
		t.Fatalf("expected 0 delay when disabled, got %v", delay)
	}
}

func TestLatencyInjector_Range(t *testing.T) {
	li := chaos.NewLatencyInjector(chaos.LatencyConfig{
		MinDelay: 20 * time.Millisecond,
		MaxDelay: 100 * time.Millisecond,
		Enabled:  true,
	})

	for i := 0; i < 10; i++ {
		delay, err := li.Delay(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if delay < 20*time.Millisecond || delay > 110*time.Millisecond {
			t.Fatalf("delay %v outside expected range", delay)
		}
	}

	stats := li.GetStats()
	if stats.DelaysApplied != 10 {
		t.Fatalf("expected 10 delays applied, got %d", stats.DelaysApplied)
	}
}

func TestLatencyInjector_Wrap(t *testing.T) {
	li := chaos.NewLatencyInjector(chaos.LatencyConfig{
		MinDelay: 10 * time.Millisecond,
		MaxDelay: 20 * time.Millisecond,
		Enabled:  true,
	})

	start := time.Now()
	called := false
	wrapped := li.Wrap(func() error {
		called = true
		return nil
	})

	if err := wrapped(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected wrapped function to be called")
	}
	if time.Since(start) < 10*time.Millisecond {
		t.Fatal("expected delay before function execution")
	}
}

// --- NetworkPartition Tests ---

func TestNetworkPartition_FullPartition(t *testing.T) {
	np := chaos.NewNetworkPartition()

	err := np.Activate(chaos.PartitionConfig{
		AffectedEndpoints: []string{"service-a:8080", "service-b:9090"},
		Mode:              chaos.PartitionFull,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !np.IsActive() {
		t.Fatal("expected partition to be active")
	}

	if !np.ShouldBlock("service-a:8080") {
		t.Fatal("expected service-a to be blocked")
	}
	if !np.ShouldBlock("service-b:9090") {
		t.Fatal("expected service-b to be blocked")
	}
	if np.ShouldBlock("service-c:7070") {
		t.Fatal("expected service-c to NOT be blocked")
	}

	stats := np.GetStats()
	if stats.RequestsBlocked != 2 {
		t.Fatalf("expected 2 blocked, got %d", stats.RequestsBlocked)
	}

	np.Deactivate()
	if np.IsActive() {
		t.Fatal("expected partition to be inactive after deactivation")
	}
}

func TestNetworkPartition_DoubleActivate(t *testing.T) {
	np := chaos.NewNetworkPartition()

	if err := np.Activate(chaos.PartitionConfig{Mode: chaos.PartitionFull}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := np.Activate(chaos.PartitionConfig{Mode: chaos.PartitionFull})
	if err == nil {
		t.Fatal("expected error on double activate")
	}

	np.Deactivate()
}

func TestNetworkPartition_AutoExpiry(t *testing.T) {
	np := chaos.NewNetworkPartition()

	err := np.Activate(chaos.PartitionConfig{
		Mode:     chaos.PartitionFull,
		Duration: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !np.IsActive() {
		t.Fatal("expected partition to be active")
	}

	time.Sleep(200 * time.Millisecond)

	if np.IsActive() {
		t.Fatal("expected partition to have expired")
	}
}

func TestNetworkPartition_Wrap(t *testing.T) {
	np := chaos.NewNetworkPartition()

	np.Activate(chaos.PartitionConfig{
		AffectedEndpoints: []string{"db:5432"},
		Mode:              chaos.PartitionFull,
	})
	defer np.Deactivate()

	called := false
	wrapped := np.Wrap("db:5432", func() error {
		called = true
		return nil
	})

	err := wrapped()
	if err == nil {
		t.Fatal("expected partition error")
	}
	if called {
		t.Fatal("expected function to NOT be called during partition")
	}
}

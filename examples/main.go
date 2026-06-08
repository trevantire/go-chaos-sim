package main

import (
	"context"
	"fmt"
	"time"

	"github.com/trevantire/go-chaos-sim/chaos"
)

func main() {
	fmt.Println("=== Chaos Simulator Demo ===\n")

	// --- Fault Injection ---
	fmt.Println("--- Fault Injection ---")

	fi := chaos.NewFaultInjector(chaos.FaultConfig{
		Type:        chaos.FaultLatency,
		Probability: 0.5,
		Delay:       200 * time.Millisecond,
		Enabled:     true,
	})

	for i := 0; i < 10; i++ {
		start := time.Now()
		err := fi.Inject(context.Background())
		elapsed := time.Since(start)
		if err != nil {
			fmt.Printf("  Call %d: fault injected (%v) in %v\n", i+1, err, elapsed)
		} else {
			fmt.Printf("  Call %d: OK in %v\n", i+1, elapsed)
		}
	}

	fmt.Printf("  Stats: %+v\n\n", fi.GetStats())

	// --- Latency Injection ---
	fmt.Println("--- Latency Injection ---")

	li := chaos.NewLatencyInjector(chaos.LatencyConfig{
		MinDelay:  50 * time.Millisecond,
		MaxDelay:  300 * time.Millisecond,
		JitterPct: 0.2,
		Enabled:   true,
	})

	for i := 0; i < 5; i++ {
		start := time.Now()
		delay, _ := li.Delay(context.Background())
		fmt.Printf("  Call %d: delayed %v (actual: %v)\n", i+1, delay, time.Since(start))
	}

	fmt.Printf("  Stats: %+v\n\n", li.GetStats())

	// --- Network Partition ---
	fmt.Println("--- Network Partition ---")

	np := chaos.NewNetworkPartition()

	fmt.Println("  Activating partition on db:5432 and cache:6379...")
	np.Activate(chaos.PartitionConfig{
		AffectedEndpoints: []string{"db:5432", "cache:6379"},
		Mode:              chaos.PartitionFull,
		Duration:          2 * time.Second,
	})

	endpoints := []string{"db:5432", "cache:6379", "api:8080"}
	for _, ep := range endpoints {
		blocked := np.ShouldBlock(ep)
		status := "ALLOWED"
		if blocked {
			status = "BLOCKED"
		}
		fmt.Printf("  %s: %s\n", ep, status)
	}

	fmt.Println("  Waiting for partition to expire...")
	time.Sleep(3 * time.Second)
	fmt.Printf("  Partition active: %v\n", np.IsActive())
	fmt.Printf("  Stats: %+v\n\n", np.GetStats())

	// --- Wrap Pattern ---
	fmt.Println("--- Wrap Pattern ---")

	wrappedDB := np.Wrap("db:5432", func() error {
		fmt.Println("  Querying database...")
		return nil
	})

	if err := wrappedDB(); err != nil {
		fmt.Printf("  DB call failed: %v\n", err)
	}

	fmt.Println("\nDone!")
}

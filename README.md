# go-chaos-sim

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A lightweight chaos engineering library for Go. Inject faults, latency, and network partitions into your code with minimal coupling.

## Features

- **Fault Injection** — Inject errors, latency, panics, and timeouts with configurable probability
- **Latency Injection** — Add realistic delays with min/max ranges and jitter
- **Network Partition** — Simulate full, partial, and blackhole network partitions between services
- **Statistics** — Track all injection activity with built-in counters
- **Zero Dependencies** — Pure Go, no external packages
- **Context-Aware** — All injectors respect `context.Context` for cancellation

## Quick Start

```go
import "github.com/trevantire/go-chaos-sim/chaos"

// Inject errors 10% of the time
fi := chaos.NewFaultInjector(chaos.FaultConfig{
    Type:        chaos.FaultError,
    Probability: 0.1,
    Enabled:     true,
    ErrorMsg:    "simulated failure",
})

// Wrap any function
safeCall := fi.Wrap(func() error {
    return doSomethingRisky()
})

if err := safeCall(); err != nil {
    log.Printf("call failed: %v", err)
}
```

## Usage

### Fault Injection

```go
fi := chaos.NewFaultInjector(chaos.FaultConfig{
    Type:        chaos.FaultLatency,
    Probability: 0.5,
    Delay:       200 * time.Millisecond,
    Enabled:     true,
})

// Inject into a call
if err := fi.Inject(ctx); err != nil {
    return err
}

// Or wrap a function
wrappedFn := fi.Wrap(myFunction)
```

**Fault types:** `FaultLatency`, `FaultError`, `FaultTimeout`, `FaultPanic`

### Latency Injection

```go
li := chaos.NewLatencyInjector(chaos.LatencyConfig{
    MinDelay:  50 * time.Millisecond,
    MaxDelay:  300 * time.Millisecond,
    JitterPct: 0.2,  // ±20% jitter
    Enabled:   true,
})

delay, err := li.Delay(ctx)
// delay = actual time slept

// Wrap with context
fn := li.WrapWithContext(ctx, func(ctx context.Context) error {
    return callService(ctx)
})
```

### Network Partition

```go
np := chaos.NewNetworkPartition()

// Activate a full partition
np.Activate(chaos.PartitionConfig{
    AffectedEndpoints: []string{"db:5432", "cache:6379"},
    Mode:              chaos.PartitionFull, // also: PartitionPartial, PartitionBlackhole
    Duration:          30 * time.Second,    // auto-expires
})

// Check before connecting
if np.ShouldBlock("db:5432") {
    return fmt.Errorf("database unreachable")
}

// Or wrap it
dbCall := np.Wrap("db:5432", func() error {
    return queryDatabase()
})
```

## API

### `FaultInjector`

| Method | Description |
|---|---|
| `NewFaultInjector(cfg)` | Create with config |
| `Inject(ctx)` | Evaluate and inject fault |
| `Wrap(fn)` | Wrap a function with fault injection |
| `UpdateConfig(cfg)` | Hot-swap config |
| `GetStats()` | Get injection statistics |

### `LatencyInjector`

| Method | Description |
|---|---|
| `NewLatencyInjector(cfg)` | Create with config |
| `Delay(ctx)` | Sleep for random duration, returns actual delay |
| `Wrap(fn)` | Add latency before function execution |
| `WrapWithContext(ctx, fn)` | Context-aware wrapper |
| `GetStats()` | Get delay statistics |

### `NetworkPartition`

| Method | Description |
|---|---|
| `NewNetworkPartition()` | Create partition simulator |
| `Activate(cfg)` | Start partition |
| `Deactivate()` | Stop partition |
| `ShouldBlock(endpoint)` | Check if endpoint is blocked |
| `Wrap(endpoint, fn)` | Wrap endpoint call |
| `IsActive()` | Check partition status |
| `GetStats()` | Get block/allow statistics |

## Examples

See [`examples/main.go`](examples/main.go) for a complete demo.

## License

MIT — see [LICENSE](LICENSE).

<!-- history: 2026-06-03 -->

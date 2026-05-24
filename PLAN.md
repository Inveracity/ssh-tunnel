# PLAN.md — ssh-tunnel Stability Improvements

## Problem

Users see spurious "use of closed network connection" errors in logs during normal connection teardown. The `pipe()` function has a race condition where each goroutine closes both connections, causing the other goroutine's `io.Copy()` to fail. Additionally, the tunnel lacks graceful shutdown, reconnection logic, and proper connection lifecycle management.

## Design Principles

- **Structs over loose functions**: Group related state (context, wait groups, config) into named structs
- **Single responsibility**: Each function does one thing and is testable in isolation
- **Error wrapping**: Use `fmt.Errorf("...: %w", err)` for error chains
- **No global state**: Pass dependencies explicitly via structs or parameters
- **Configurable defaults**: Use an options pattern or config struct for tunable values (retry counts, backoff, timeouts)

## Architecture

### New Types

```go
// internal/tunnel/tunnel.go

type Tunnel struct {
    User     string
    Local    Local
    Remote   Remote
}

type Local struct {
    Port string
    Cmd  []string // optional post-connect command
}

type Remote struct {
    Port string
    Host string
}

type Config struct {
    MaxRetries      int           // max SSH reconnect attempts (0 = unlimited)
    RetryBaseDelay  time.Duration // initial backoff (default: 1s)
    RetryMaxDelay   time.Duration // max backoff cap (default: 30s)
    AcceptRetryDelay time.Duration // delay on transient accept errors (default: 100ms)
}

type TunnelRunner struct {
    tunnel Tunnel
    config Config
    ctx    context.Context
    cancel context.CancelFunc
    wg     sync.WaitGroup // tracks active pipe goroutines
    mu     sync.Mutex     // protects listener lifecycle
}
```

### Method Breakdown

| Method | Responsibility |
|--------|---------------|
| `NewTunnelRunner(ctx, tunnel, config)` | Constructor, creates cancellable child context |
| `Run()` | Entry point: connects to SSH agent, builds SSH config, starts accept loop |
| `connectSSH()` | Establishes SSH connection with retry logic |
| `runAcceptLoop(conn)` | Accepts connections, spawns pipes, handles transient errors |
| `acceptConnection(listener)` | Single accept with context awareness |
| `dialRemote(conn, remote)` | Dials remote through SSH tunnel |
| `spawnPipePair(local, remote)` | Creates bidirectional pipe with shared `sync.Once` and `sync.WaitGroup` tracking |
| `pipe(ctx, writer, reader, closeOnce)` | Unidirectional data copy with context cancellation |
| `Close()` | Graceful shutdown: stops listener, waits for pipes, closes SSH |

## Changes

### 1. Fix `pipe()` race condition with `sync.Once`

- `pipe()` receives a `*sync.Once` shared between both directions
- Only the first goroutine to finish closes both connections
- Errors from `io.Copy()` are still logged (no suppression)
- `pipe()` respects context cancellation via `io.Copy` interruption

### 2. Handle `listener.Accept()` errors gracefully

- Distinguish transient errors from permanent ones
- On transient errors, sleep briefly (configurable) and continue
- Use context cancellation to break the accept loop cleanly
- Extract accept logic into `acceptConnection()` for clarity

### 3. Add SSH reconnection logic

- `connectSSH()` wraps `ssh.Dial()` with exponential backoff
- Backoff respects context cancellation (use `time.After` + `select`)
- Configurable max retries and delay bounds
- On reconnection, the accept loop continues with the new connection

### 4. Add context/cancellation for graceful shutdown

- `TunnelRunner` holds a cancellable child context
- `main.go` listens for SIGINT/SIGTERM and calls `cancel()`
- `listener.Accept()` is interruptible via closing the listener or using a deadline-based approach
- `pipe()` checks `ctx.Done()` to abort mid-copy

### 5. Add connection tracking with `sync.WaitGroup`

- `TunnelRunner.wg` tracks all active `pipe()` goroutines
- `spawnPipePair()` calls `wg.Add(1)` per pipe direction
- `Close()` calls `wg.Wait()` after stopping the listener
- Prevents goroutine leaks and ensures clean teardown

## Implementation Order

Each step should compile and pass `go vet ./...` independently.

### Step 1: Define new types and constructor
- Create `Config`, `TunnelRunner` structs
- Add `NewTunnelRunner(ctx, tunnel, config)` constructor
- Keep existing `Start()` as a thin wrapper for backward compatibility with `main.go`

### Step 2: Extract `pipe()` with `sync.Once` and context
- Refactor `pipe()` to accept `context.Context` and `*sync.Once`
- Add context-aware copy (check `ctx.Done()` or use `io.Copy` with cancel-aware reader)
- `spawnPipePair()` method creates the shared `sync.Once` and spawns both directions

### Step 3: Extract `runAcceptLoop()` with `sync.WaitGroup` tracking
- Move accept loop into `TunnelRunner.runAcceptLoop(conn)`
- Track pipes via `TunnelRunner.wg`
- Extract `acceptConnection()` and `dialRemote()` methods

### Step 4: Add graceful shutdown via `Close()`
- Implement `TunnelRunner.Close()` — closes listener, waits for pipes, closes SSH
- Wire context cancellation to trigger `Close()`
- Add `defer runner.Close()` pattern in `Run()`

### Step 5: Handle `listener.Accept()` errors gracefully
- Add transient error detection in `runAcceptLoop()`
- Use `AcceptRetryDelay` config for backoff on transient errors
- Context cancellation breaks the loop

### Step 6: Add signal handling in `main.go`
- Listen for SIGINT/SIGTERM via `os/signal`
- Cancel context on signal
- Wait for all tunnel goroutines to finish before exiting

### Step 7: Add SSH reconnection logic
- Implement `connectSSH()` with exponential backoff
- Wrap accept loop in a reconnect outer loop
- On SSH disconnect, attempt reconnect before giving up
- Respect context cancellation during backoff

### Step 8: Update `main.go` to use new API
- Replace direct `tunnel.Start()` call with `NewTunnelRunner()` + `Run()`
- Pass signal-aware context

### Step 9: Final lint pass
- Run `go vet ./...`
- Run `go fmt ./...`
- Run `golangci-lint run ./...` if available

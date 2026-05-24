# PLAN.md — ssh-tunnel Maintenance Refactor

## Issues Found

### Bugs
1. `main.go:62-63` — `defer wg.Done()` inside a for loop is a **deadlock**. Defer runs when `run()` returns, but `run()` is blocked on `wg.Wait()`. Should be `go func() { defer wg.Done(); tunnel.Start(c) }()`.
2. `config.go:81` — `os.UserHomeDir()` error silently ignored.

### Lint (9 errcheck violations)
- `main.go:67` — unchecked `exec.Command().Run()`
- `config.go:63,73,75,94` — unchecked `fmt.Fprintf()`, `w.Flush()`, `green.Println()`
- `template.go:15` — unchecked `os.WriteFile()`
- `tunnel.go:30,66,67` — unchecked `conn.Close()`, `writer.Close()`, `reader.Close()`

### Code Quality
- `config.go:Parse()` uses `log.Fatal` — should return errors for testability.
- `tunnel.go:Start()` uses `log.Fatalln` — same issue.
- `tunnel.go:tunnel()` has nested goroutines + if/else that can be flattened.
- `tunnel.go` imports `config` directly — should define its own `Tunnel` struct or interface.
- `config.go:cwd()` uses `filepath.Abs("./")` instead of `os.Getwd()`.
- No `.golangci.yml` config file.
- CI workflow uses Go 1.20 but `go.mod` requires 1.26.3.

---

## Incremental Plan

### Step 1: Add `.golangci.yml` and fix CI Go version
- Create `.golangci.yml` with sensible defaults for this project
- Update `.github/workflows/ci.yaml` Go version to `1.26`

### Step 2: Fix the deadlock bug in `main.go`
- Move `wg.Done()` into the goroutine properly

### Step 3: Fix all errcheck violations
- Handle or explicitly ignore all unchecked error returns across all files

### Step 4: Refactor `config.go` — return errors instead of `log.Fatal`
- `Parse()` returns `([]Tunnel, error)` instead of calling `log.Fatal`
- `FindConfig()` already returns error — fix the ignored `UserHomeDir()` error
- Push error handling up to `main.go`

### Step 5: Refactor `tunnel.go` — return errors, flatten nesting
- `Start()` returns error instead of `log.Fatalln`
- Extract `pipe()` from inside `tunnel()` to package level
- Flatten the if/else in the accept loop
- Define a local `Tunnel` struct in `tunnel` package to break the `config` import dependency

### Step 6: Refactor `template.go` — handle errors
- `Write()` returns error
- `main.go` handles it

### Step 7: Final lint pass
- Run `golangci-lint run ./...` clean
- Run `go vet ./...` clean
- Run `go fmt ./...`

---

Each step should compile and pass lint independently.

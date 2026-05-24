# AGENTS.md

## Project

Single-binary Go CLI (`github.com/inveracity/ssh-tunnel`). Opens SSH tunnels defined in an HCL config file.

## Commands

- `make ssh-tunnel` — build binary to `./ssh-tunnel`
- `make install` — build and install to `~/.local/bin/ssh-tunnel`
- `make zip VERSION=x.y.z` — build + zip for release
- `./ssh-tunnel --init` — generate example `ssh-tunnel.hcl` config
- `./ssh-tunnel --config <file>` — use non-default config (default: `ssh-tunnel.hcl`)

## Architecture

- `cmd/ssh-tunnel/main.go` — entrypoint, flag parsing, goroutine-per-tunnel
- `internal/config/` — HCL config parsing and file discovery
- `internal/tunnel/` — SSH tunnel execution (uses `SSH_AUTH_SOCK`)
- `internal/template/` — config file generation (`--init`)
- `internal/version/` — version string, injected via ldflags at build

## Key Constraints

- **Requires a running SSH agent** (`SSH_AUTH_SOCK` env var). Will not work without it.
- **Linux only.** Not tested on macOS; does not work on Windows (except WSL2).
- **No tests exist.** Do not expect a test suite to run.
- **Linting:** `golangci-lint run ./...` — configured via `.golangci.yml` (add if missing). `go vet` and `go fmt` are safe defaults.

## CI

`.github/workflows/ci.yaml` — release-only (triggered on GitHub release or manual dispatch). Builds with `make ssh-tunnel VERSION=<tag>`, zips, uploads artifact.

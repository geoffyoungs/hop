# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

hop is a Go CLI tool for connecting to named hosts using an INI configuration file (~/.config/hop/hosts.ini). It supports SSH, Docker, and Kubernetes backends.

## Build Commands

```bash
go build ./cmd/hop           # Build binary
go test ./...                # Run tests
go run ./cmd/hop --help      # Run without building
goreleaser build --snapshot --clean  # Build release packages
```

## Architecture

```
cmd/hop/main.go              # CLI entry point (Cobra commands)
internal/
  backend/
    backend.go               # Backend interface definition
    registry.go              # Backend registration system
    ssh.go                   # SSH backend (wraps ssh/scp)
    docker.go                # Docker backend (wraps docker exec/cp)
    k8s.go                   # Kubernetes backend (wraps kubectl)
  config/
    config.go                # INI file loading from XDG config path
    host.go                  # HostConfig struct and conversion
```

## Key Patterns

- **Backend interface**: All connection types implement `Backend` interface with Connect, Copy, ForwardPort, Check, Validate methods
- **Registry pattern**: Backends self-register via `init()` functions
- **XDG config**: Config loaded from `~/.config/hop/hosts.ini` by default
- **Cobra CLI**: Standard subcommand structure with completion support

## Adding New Backends

1. Create `internal/backend/newbackend.go`
2. Implement the `Backend` interface
3. Add `init()` function that calls `Register(&NewBackend{})`
4. The backend will be automatically available

## Releasing

Releases MUST be done using goreleaser. This is the only supported way to release.

```bash
# 1. Ensure working tree is clean
git status

# 2. Create and push a version tag
git tag -a v0.x.0 -m "v0.x.0 - Release description"
git push origin v0.x.0

# 3. Run goreleaser (uses gh auth token)
GITHUB_TOKEN=$(gh auth token) goreleaser release --clean
```

This will:
- Build binaries for darwin/linux (amd64/arm64)
- Generate shell completions (bash/zsh/fish)
- Create .tar.gz, .deb, and .rpm packages
- Upload to GitHub releases
- Update the Homebrew tap

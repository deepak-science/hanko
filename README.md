# hanko

> 判子 — the stamp you press onto a finished thing.

`hanko` is a small Go CLI that computes a version from your git history and
stamps it onto build artifacts: container images, helm charts, Go binaries,
OS packages, archives. It is intended as a Nix-friendly, single-static-binary
replacement for [GitVersion](https://github.com/GitTools/GitVersion).

## Status

Skeleton. See [ROADMAP.md](./ROADMAP.md) for the path to v1.

## Quick start

```sh
nix build
./result/bin/hanko version             # → 0.0.0  (placeholder)
./result/bin/hanko version --format json
./result/bin/hanko version --format env
```

## Commands (planned)

| Command                       | Purpose                                                   |
| ----------------------------- | --------------------------------------------------------- |
| `hanko version`               | Compute the current version. Formats: semver/full/json/env |
| `hanko tag`                   | Create (and optionally push) a git tag for that version   |
| `hanko stamp docker <image>`  | Apply OCI `org.opencontainers.image.*` labels             |
| `hanko stamp helm <chart>`    | Set `version` and `appVersion` in `Chart.yaml`            |
| `hanko stamp go-ldflags`      | Emit `-ldflags` for stamping a Go binary at build time    |

## Build & develop

This repo uses Nix + `gomod2nix`. Common tasks live in the `justfile`:

- `just build` — build the binary via `nix build`
- `just test`  — run Go tests in the devshell
- `just check` — `nix flake check`
- `just chores` — `go mod tidy` + regenerate `gomod2nix.toml`

## Layout

```
.
├── main.go
├── cmd/                # cobra commands
├── internal/
│   ├── gitinfo/        # extract relevant git state
│   ├── version/        # version-calculation engine
│   └── logging/        # slog file logger
├── flake.nix
├── justfile
└── ROADMAP.md
```

# hanko

> 判子 — the stamp you press onto a finished thing.

`hanko` is a small Go CLI that computes a version from your git history and stamps it onto build artifacts: container images, helm charts, Go binaries, OS packages, archives.
It is intended as a more specific, single-static-binary replacement for [GitVersion](https://github.com/GitTools/GitVersion).

## Philosophy

Hanko has three main commands:
- `hanko version`: return a descriptor of current repository state. read-only and idempotent.
- `hanko version` — *what is this?* Read-only. Idempotent. Same `(commit, branch, dirty, tag-history)` → same answer. Re-run it freely; every CI job that needs the label just re-computes from its own checkout.
- `hanko stamp …`  — *apply this commit's identity to artifact X.* Writes the computed version into Chart.yaml, ldflags, Docker labels, etc.
- `hanko tag`      — *promote this commit's identity to a permanent git ref.* The only release-shaped act, and even then it's persisting an identity hanko already computed — it doesn't decide the version, git state does.

A useful litmus test: if running `hanko version` could change behavior elsewhere, the design is wrong.
It's a label-reader, not a state-machine step.

## Status

M0–M3 shipped: real version computation, idempotent tagging, and stamping for `go-ldflags` / `docker tags` / `docker labels` / `helm` work end-to-end against unit, smoke, and flow tests.
See [ROADMAP.md](./ROADMAP.md) for what's left before v1, and [docs/design-decisions.md](./docs/design-decisions.md) for open design questions.

## Quick start

```sh
nix build
./result/bin/hanko version             # → e.g. 1.2.3 or 1.2.3-feature-foo.4
./result/bin/hanko version --format full
./result/bin/hanko version --format json
./result/bin/hanko version --format env
./result/bin/hanko version --format gha  # key=value lines for $GITHUB_OUTPUT
```

For more, see [examples/local-usage.md](./examples/local-usage.md) and the migration sketches in [examples/migrations/](./examples/migrations/).

## Commands

| Command                              | Purpose                                                                |
| ------------------------------------ | ---------------------------------------------------------------------- |
| `hanko version`                      | Compute the current version. Formats: `semver` / `full` / `json` / `env` / `gha` |
| `hanko tag [--push]`                 | Create (and optionally push) an annotated git tag for that version     |
| `hanko stamp go-ldflags`             | Emit `-X main.version=… -X main.commit=… -X main.date=…` for `go build` |
| `hanko stamp docker tags <image>`    | Fan version out into `<image>:<full>`, `:<major.minor>`, `:<major>`, `:latest`, `:<branch>-<sha>` |
| `hanko stamp docker labels`          | Emit `org.opencontainers.image.*` labels as `--label` args or a label file |
| `hanko stamp helm <chart-dir>`       | Set `version` and `appVersion` in `Chart.yaml` in place                |

## Build & develop

This repo uses Nix + `gomod2nix`.
Common tasks live in the `justfile`:

- `just build` — build the binary via `nix build`
- `just test` — run Go unit tests
- `just smoke` — CLI smoke tests on minimal repos (verifies command shape, flag handling, exit codes)
- `just flows` — CLI flow tests on mock repos with realistic tag histories (verifies outcomes on hotfix / release-branch / multi-tag / push-to-remote scenarios)
- `just check-cli` — both `smoke` and `flows`
- `just check` — `nix flake check`
- `just fmt` — format files via treefmt
- `just fixtures` — (re)build dev fixtures under `./fixtures/` (gitignored)
- `just chores` — `go mod tidy` + regenerate `gomod2nix.toml`


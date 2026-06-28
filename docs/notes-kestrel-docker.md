# Notes from wiring kestrel's docker publish (2026-06-12)

Feedback from the first real consumer of `hanko version docker tags`
(kestrel's `scripts/deploy-image.sh`, ghcr multi-arch publish on `v*` tags).
Both are patch candidates, neither blocked the integration.

> **Status (2026-06-25):** all three addressed on `stamp-improvements`. See
> design-decisions D-020 (items 1 & 2) and D-021 (the nix note).

## 1. `:latest` is never emitted from a tag-push CI checkout

`--latest-on-default-branch` only fires when `BranchName` is `main`/`master`,
but the canonical release trigger (`on: push: tags:`) checks out a detached
HEAD, so `BranchName` is `"detached"` even though D-001 makes the *version*
come out right. Net effect: the one place you most want `:latest` — a release
publish job — can't get it without a workaround.

- Kestrel's workaround: `hanko version docker tags "$IMAGE" --extra latest`,
  which silently breaks the "non-prerelease only" guard that
  `--latest-on-default-branch` normally provides.
- Possible fix: when detached-at-tag (the D-001 special case) and the version
  is non-prerelease, treat it as latest-eligible — optionally only if the tag
  is reachable from the default branch (`git merge-base --is-ancestor`).
- **Done (D-020):** `--latest-on-release-tag` (default false). A D-001 build now
  carries `Version.AtReleaseTag`; the flag makes it `:latest`-eligible, still
  guarded by non-prerelease. Kestrel drops `--extra latest` for
  `--latest-on-release-tag` and gets the guard back. Reachability-gating was
  considered and parked (would force a git dep into the pure emitter).

## 2. No way to emit *only* the branch-sha ref

`--branch-sha-tag=false` suppresses `<branch>-<sha>`, but there's no inverse —
no flag to suppress the semver fan-out (`X.Y.Z`, `X.Y`, `X`) and keep just the
branch-sha ref. That's the natural shape for non-release mainline "edge"
publishing (push `main-abc1234` on every main commit, semver refs only on
release tags). Kestrel sidestepped this by gating image publishing to `v*`
tags entirely.

- Possible fix: `--semver-tags=false`, or a `--only branch-sha` selector.
- **Done (D-020):** `--semver-tags=false` suppresses the `<full>/<major.minor>/
  <major>/:latest` fan-out, leaving just `<branch>-<sha>` (and `--extra`s).

## Minor observation (no change needed, maybe a docs note)

Nix-built images (`dockerTools.buildLayeredImage`) can't consume
`hanko version docker labels` at build time — there's no `docker build` step,
and the nix sandbox can't run hanko anyway. Kestrel bakes the OCI labels into
the flake from the hanko-sealed `version` variable instead. Worth a line in
the docker-emitter docs so the next nix consumer doesn't go looking for a
stamp target that can't exist.

- **Done (D-021):** documented in the README build-time-emitter notes.

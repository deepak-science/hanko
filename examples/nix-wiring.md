# Nix wiring

How to consume hanko from another flake — both the `hanko` CLI (devshells, CI) and the `hanko.lib` helpers (build-time ldflag wiring).

## Add hanko as a flake input

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    hanko = {
      url = "github:dmallubhotla/hanko";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };
}
```

## CLI on PATH (devshells, `just release`)

```nix
devShells.default = pkgs.mkShell {
  buildInputs = [
    hanko.packages.${pkgs.system}.default
    # or: bring in the overlay and use `pkgs.hanko`
  ];
};
```

The overlay (`hanko.overlays.default`) registers `pkgs.hanko` and re-exports gomod2nix's `buildGoApplication`.

## Stamped version pattern (D-015)

Hoist `version` into a `let` binding so `hanko stamp nix` has one obvious line to rewrite, and downstream derivations `inherit version`:

```nix
let
  # Stamped by hanko; do not hand-edit.
  version = "0.0.0";
in
{
  packages.default = pkgs.buildGoApplication {
    pname = "my-app";
    inherit version;
    src = ./.;
    modules = ./gomod2nix.toml;
    ldflags = hanko.lib.mkGoLdflags { inherit self version; };
  };
}
```

`.hanko.yaml` declares the stamp target:

```yaml
stamp-targets:
  - path: flake.nix
    format: nix
    key: version
```

## `hanko.lib.mkGoLdflags`

Pure-nix equivalent of `hanko version go-ldflags`. Emits the same `-X main.version=… -X main.commit=… -X main.date=…` shape, sourced from the stamped `version` binding plus the flake's `self.rev` / `self.lastModifiedDate` — no git or hanko required inside the build sandbox.

```nix
hanko.lib.mkGoLdflags {
  inherit self;            # for commit + date
  version = "1.2.3";       # the stamped value
  package = "main";        # optional; defaults to "main"
  strip = true;            # optional; prepends "-s" "-w" (default true)
}
# → [ "-s" "-w" "-X" "main.version=1.2.3" "-X" "main.commit=…" "-X" "main.date=…" ]
```

### Why not just call `hanko version go-ldflags`?

It works fine from a devshell or CI step, but inside a nix build the sandbox has no git history to compute from. The pure-nix helper sidesteps that — it reads the stamped `version` (already correct because release-time stamping happened) and pulls `commit` / `date` from flake metadata, which nix has even in the sandbox.

### Semantic differences from the CLI form

- **Version string.** CLI emits the *computed* SemVer (branch-suffixed off mainline, e.g. `1.2.3-feature-foo.4`); the helper emits the *stamped* value (typically the latest release). For release builds this is identical; for feature-branch builds the binary will report the last release rather than a branch-specific string.
- **Date format.** CLI emits ISO-8601 (`2026-06-01T05:26:00-05:00`); the helper emits `self.lastModifiedDate`'s compact form (`YYYYMMDDHHMMSS`).
- **Strip flags.** CLI does not emit `-s -w`; the helper does by default. Pass `strip = false` to match CLI output exactly.

If branch-suffixed version strings inside nix-built binaries matter to you, run `hanko version go-ldflags` in a CI step that builds outside the sandbox, or open an issue — there's no second consumer driving that shape yet.

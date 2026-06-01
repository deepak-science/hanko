{
  description = "hanko CLI dev environment";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };
  outputs =
    {
      self,
      nixpkgs,
      gomod2nix,
      ...
    }@inputs:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];
      eachSystem = f: nixpkgs.lib.genAttrs supportedSystems (system: f (pkgsFor system));
      pkgsFor =
        system:
        let
          pkgs = import nixpkgs {
            inherit system;
            config.allowUnfree = true;
          };
        in
        pkgs.extend (
          nixpkgs.lib.composeManyExtensions [
            gomod2nix.overlays.default
            hankoOverlay
          ]
        );
      treefmtEval = eachSystem (pkgs: inputs.treefmt-nix.lib.evalModule pkgs ./treefmt.nix);
      # Stamped by hanko; do not hand-edit (use `just release`).
      # Hoisted out of the overlay so it's a clear single source of truth even
      # though hanko currently only exposes one derivation — matches the D-015
      # shared-`let` pattern recommended for consumers.
      version = "0.3.0";
      # Pure-nix equivalent of `hanko version go-ldflags`. Exposed as
      # `hanko.lib.mkGoLdflags` so consumers can wire ldflags into their own
      # buildGoApplication / buildGoModule without shelling out to hanko at
      # build time (the build sandbox lacks git, and depending on a built
      # hanko to build hanko is circular).
      hankoLib = {
        mkGoLdflags =
          {
            self,
            version,
            package ? "main",
            strip ? true,
          }:
          (
            if strip then
              [
                "-s"
                "-w"
              ]
            else
              [ ]
          )
          ++ [
            "-X"
            "${package}.version=${version}"
            "-X"
            "${package}.commit=${self.rev or self.dirtyRev or "unknown"}"
            "-X"
            "${package}.date=${self.lastModifiedDate or "unknown"}"
          ];
      };
      commonLdflags = hankoLib.mkGoLdflags { inherit self version; };
      hankoOverlay = final: _prev: {
        hanko = final.buildGoApplication {
          pname = "hanko";
          inherit version;
          src = ./.;
          modules = ./gomod2nix.toml;
          nativeBuildInputs = [ final.installShellFiles ];
          nativeCheckInputs = [ final.git ];
          ldflags = commonLdflags;
          postInstall = ''
            installShellCompletion --cmd hanko \
              --bash <($out/bin/hanko completion bash) \
              --zsh <($out/bin/hanko completion zsh) \
              --fish <($out/bin/hanko completion fish)
          '';
        };
      };
    in
    {
      lib = hankoLib;

      overlays.default = nixpkgs.lib.composeManyExtensions [
        gomod2nix.overlays.default
        hankoOverlay
      ];

      packages = eachSystem (pkgs: {
        default = pkgs.hanko;
      });

      formatter = eachSystem (pkgs: treefmtEval.${pkgs.stdenv.hostPlatform.system}.config.build.wrapper);
      checks = eachSystem (pkgs: {
        formatting = treefmtEval.${pkgs.stdenv.hostPlatform.system}.config.build.check self;
        smoke =
          pkgs.runCommand "hanko-smoke"
            {
              nativeBuildInputs = [ pkgs.git ];
            }
            ''
              export HOME=$TMPDIR
              bash ${./test/smoke/smoke.sh} ${pkgs.hanko}/bin/hanko
              touch $out
            '';
        # Lint as a check: override the hanko derivation to swap go test for
        # golangci-lint in checkPhase. Reuses the vendored module setup from
        # goConfigHook so no network is needed inside the sandbox.
        golangci-lint = pkgs.hanko.overrideAttrs (old: {
          pname = "hanko-golangci-lint";
          nativeCheckInputs = (old.nativeCheckInputs or [ ]) ++ [ pkgs.golangci-lint ];
          doCheck = true;
          checkPhase = ''
            runHook preCheck
            export GOLANGCI_LINT_CACHE=$TMPDIR/golangci-lint-cache
            golangci-lint run --timeout 5m ./...
            runHook postCheck
          '';
        });
      });

      devShells = eachSystem (pkgs: {
        default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gotools
            golangci-lint
            go-tools
            gopls
            gomod2nix.packages.${pkgs.stdenv.hostPlatform.system}.default
            # Hanko-on-itself: lets `just release` invoke `hanko stamp nix`
            # without `go run .`. If hanko's source breaks the build, fall
            # back to `nix develop --command go run . ...` from outside the
            # devshell.
            pkgs.hanko
          ];
          shellHook = ''
            echo "hanko devshell"
          '';
        };
      });
    };
}

package smoke

import (
	"strings"
	"testing"

	"github.com/dmallubhotla/hanko/internal/testrepo"
)

const noPush = "seal:\n  push-remote: \"\"\n"

// sealRepo is a repo with a committed no-push .hanko.yaml.
func sealRepo(t *testing.T) *testrepo.Builder {
	t.Helper()
	return testrepo.New(t).Commit("one").WriteFile(".hanko.yaml", noPush).CommitAll("add hanko config")
}

func tagList(r *testrepo.Builder) string {
	return strings.ReplaceAll(git(r, "tag", "-l"), "\n", ",")
}

func TestSeal(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).Commit("one").
		WriteFile(".hanko.yaml", noPush+"  pre-commit:\n    - \"echo hook saw {semver}\"\n").
		CommitAll("add hanko config").
		Tag("v0.1.0").Commit("two")
	dir := r.Dir()

	t.Run("dry-run", func(t *testing.T) {
		out := ok(t, dir, "seal", "--dry-run")
		contains(t, "shows version", "version:        0.1.1", out)
		contains(t, "shows tag", "tag name:       v0.1.1", out)
		contains(t, "shows default commit message with chore prefix", "chore: Release 0.1.1", out)
		contains(t, "expands hook template", "echo hook saw 0.1.1", out)
		eq(t, "created no tag", "v0.1.0", tagList(r))
	})

	t.Run("happy path, local", func(t *testing.T) {
		contains(t, "prints created tag", "v0.1.1", hanko(t, dir, "seal").combined)
		eq(t, "tag v0.1.1 was created", "v0.1.1", git(r, "tag", "-l", "v0.1.1"))
	})
}

func TestSeal_stampsBeforeHooks(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).
		WriteFile("pyproject.toml", pyproject).
		WriteFile(".hanko.yaml", noPush+"stamp-targets:\n  - path: pyproject.toml\n    format: toml\n    key: project.version\n").
		CommitAll("initial").Tag("v1.0.0").Commit("two")

	ok(t, r.Dir(), "seal")
	eq(t, "seal bumped pyproject", `version = "1.0.1"`, lineWithPrefix(t, r, "pyproject.toml", "version"))
	eq(t, "seal created tag", "v1.0.1", git(r, "tag", "-l", "v1.0.1"))
	contains(t, "release commit touched pyproject", "pyproject.toml", git(r, "show", "--name-only", "--pretty=format:", "HEAD"))
}

// Bare repo as origin: seal must push the release commit and tag together.
func TestSeal_pushesToBareOrigin(t *testing.T) {
	t.Parallel()
	bare := testrepo.NewBare(t)
	r := testrepo.New(t).
		WriteFile("flake.nix", "{\n  outputs = _: {\n    packages.default = mkDerivation {\n      version = \"0.0.1\";\n    };\n  };\n}\n").
		WriteFile(".hanko.yaml", "stamp-targets:\n  - path: flake.nix\n    format: nix\n    key: version\nseal:\n  commit-message: \"Release {semver}\"\n").
		CommitAll("add flake").Tag("v1.0.0").Commit("two")
	r.Git("remote", "add", "origin", bare)

	contains(t, "seal output names the tag", "v1.0.1", hanko(t, r.Dir(), "seal").combined)

	head := git(r, "rev-parse", "HEAD")
	eq(t, "tag pushed to bare origin", "v1.0.1", git(r, "-C", bare, "tag", "-l", "v1.0.1"))
	eq(t, "tag in origin points at the release commit", head, git(r, "-C", bare, "rev-list", "-n1", "v1.0.1"))
	eq(t, "main updated in origin", head, git(r, "-C", bare, "rev-parse", "main"))
	contains(t, "flake.nix in release commit was bumped", `version = "1.0.1"`, git(r, "-C", bare, "show", "v1.0.1:flake.nix"))
}

// With `increment: none` and no feat:/fix: signal, the computed tag is the one we already have.
func TestSeal_noReleaseNeeded(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).Commit("one").
		WriteFile(".hanko.yaml", noPush+`branches:
  - name: mainline
    regex: ^(main|master)$
    is-mainline: true
    increment: none
    label: ""
  - name: feature
    regex: .*
    increment: none
    label: "{branch}"
`).
		CommitAll("add hanko config").
		Tag("v1.0.0").Commit("chore: tidy").Commit("docs: typo")

	res := hanko(t, r.Dir(), "seal")
	if res.code != 0 {
		t.Errorf("want exit 0 on no-release-needed, got %d", res.code)
	}
	contains(t, "names the no-release path", "no release needed", res.combined)
	contains(t, "identifies the unchanged tag", "v1.0.0", res.combined)
	contains(t, "hints at why (no signal)", "none with feat", res.combined)
	eq(t, "no new tag was created", "v1.0.0", tagList(r))
}

func TestSeal_refusesDirty(t *testing.T) {
	t.Parallel()
	r := sealRepo(t).Tag("v0.1.0").Commit("two").WriteFile("dirt", "dirt\n")
	contains(t, "error mentions dirty", "dirty", fails(t, r.Dir(), "seal"))
}

func TestSeal_refusesPrerelease(t *testing.T) {
	t.Parallel()
	r := sealRepo(t).Tag("v0.1.0").Checkout("feature/x").Commit("two")
	contains(t, "error mentions pre-release", "pre-release", fails(t, r.Dir(), "seal"))
}

func TestSeal_initial(t *testing.T) {
	t.Parallel()

	t.Run("verbatim value", func(t *testing.T) {
		t.Parallel()
		r := sealRepo(t)
		contains(t, "seal output names the tag", "v0.1.0", hanko(t, r.Dir(), "seal", "--initial=v0.1.0").combined)
		eq(t, "tag v0.1.0 was created", "v0.1.0", git(r, "tag", "-l", "v0.1.0"))
	})

	t.Run("bare form uses initial-version from config", func(t *testing.T) {
		t.Parallel()
		r := testrepo.New(t).Commit("one").
			WriteFile(".hanko.yaml", "initial-version: \"0.2.0\"\n"+noPush).
			CommitAll("add hanko config")
		contains(t, "seal output names the configured tag", "0.2.0", hanko(t, r.Dir(), "seal", "--initial").combined)
		eq(t, "bare tag 0.2.0 was created", "0.2.0", git(r, "tag", "-l", "0.2.0"))
	})

	t.Run("stamps the initial value into stamp-targets", func(t *testing.T) {
		t.Parallel()
		r := testrepo.New(t).
			WriteFile("pyproject.toml", "[project]\nname = \"demo\"\nversion = \"0.0.0\"\n").
			WriteFile(".hanko.yaml", noPush+"stamp-targets:\n  - path: pyproject.toml\n    format: toml\n    key: project.version\n").
			CommitAll("initial")
		ok(t, r.Dir(), "seal", "--initial=v0.1.0")
		eq(t, "stamp-target received the semver without v", `version = "0.1.0"`, lineWithPrefix(t, r, "pyproject.toml", "version"))
	})

	t.Run("refuses if a semver tag exists", func(t *testing.T) {
		t.Parallel()
		r := testrepo.New(t).Commit("one").Tag("v0.1.0").
			WriteFile(".hanko.yaml", noPush).CommitAll("add hanko config").Commit("two")
		contains(t, "error explains why", "--initial only valid when no semver-shaped tag exists", fails(t, r.Dir(), "seal", "--initial=v1.0.0"))
	})

	t.Run("refuses non-semver values", func(t *testing.T) {
		t.Parallel()
		contains(t, "error mentions shape", "not a semver-shaped tag", fails(t, sealRepo(t).Dir(), "seal", "--initial=banana"))
	})

	t.Run("refuses combination with --bump", func(t *testing.T) {
		t.Parallel()
		contains(t, "error mentions mutually exclusive", "mutually exclusive", fails(t, sealRepo(t).Dir(), "seal", "--initial=v0.1.0", "--bump=minor"))
	})

	t.Run("prerelease-shaped initial still trips the blocker", func(t *testing.T) {
		t.Parallel()
		contains(t, "error mentions pre-release", "pre-release", fails(t, sealRepo(t).Dir(), "seal", "--initial=v0.1.0-beta.1"))
	})
}

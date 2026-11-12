package smoke

import (
	"testing"

	"github.com/dmallubhotla/hanko/internal/testrepo"
)

const pyproject = `[project]
name = "demo"
version = "0.0.1"
`

func TestStamp_declarativeTargets(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).
		WriteFile("pyproject.toml", pyproject).
		WriteFile("package.json", "{\n  \"name\": \"demo\",\n  \"version\": \"0.0.1\"\n}\n").
		WriteFile("Chart.yaml", "apiVersion: v2\nname: demo\nversion: 0.0.1\nappVersion: \"0.0.1\"\n").
		WriteFile(".hanko.yaml", `seal:
  push-remote: ""
stamp-targets:
  - path: pyproject.toml
    format: toml
    key: project.version
  - path: package.json
    format: json
    key: version
  - path: Chart.yaml
    format: yaml
    keys: [version, appVersion]
`).
		CommitAll("initial").Tag("v1.0.0").Commit("two")
	dir := r.Dir()

	t.Run("dry-run", func(t *testing.T) {
		out := ok(t, dir, "stamp", "--dry-run")
		contains(t, "lists pyproject", "pyproject.toml (toml)", out)
		contains(t, "lists package.json", "package.json (json)", out)
		contains(t, "lists Chart.yaml", "Chart.yaml (yaml)", out)
		contains(t, "shows the new value in pyproject", "project.version: 0.0.1 → 1.0.1", out)
		eq(t, "did not write pyproject", `version = "0.0.1"`, lineWithPrefix(t, r, "pyproject.toml", "version"))
	})

	t.Run("apply", func(t *testing.T) {
		ok(t, dir, "stamp")
		eq(t, "pyproject bumped", `version = "1.0.1"`, lineWithPrefix(t, r, "pyproject.toml", "version"))
		contains(t, "package.json bumped", `"version": "1.0.1"`, r.ReadFile("package.json"))
		eq(t, "Chart.yaml version bumped", "version: 1.0.1", lineWithPrefix(t, r, "Chart.yaml", "version:"))
		eq(t, "Chart.yaml appVersion bumped (quoted preserved)", `appVersion: "1.0.1"`, lineWithPrefix(t, r, "Chart.yaml", "appVersion:"))
	})
}

func TestStamp_refusesWithoutTargets(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).Commit("one")
	contains(t, "error mentions stamp-targets", "stamp-targets", fails(t, r.Dir(), "stamp"))
}

// `stamp <format>` is a positional filter on stamp-targets; the old standalone one-off modes are gone.
func TestStamp_nixFilter(t *testing.T) {
	t.Parallel()
	const flake = `{
  outputs = _: {
    packages.default = mkDerivation {
      pname = "demo";
      version = "0.0.1";
      src = ./.;
    };
  };
}
`
	r := testrepo.New(t).
		WriteFile("flake.nix", flake).
		WriteFile(".hanko.yaml", "stamp-targets:\n  - path: flake.nix\n    format: nix\n    key: version\n").
		CommitAll("one").Tag("v1.2.3").Commit("two")
	dir := r.Dir()

	ok(t, dir, "stamp", "nix")
	contains(t, "flake.nix version bumped", `version = "1.2.4";`, r.ReadFile("flake.nix"))

	r.Git("checkout", "-q", "--", "flake.nix")
	contains(t, "dry-run announces change", "0.0.1 → 1.2.4", ok(t, dir, "stamp", "nix", "--dry-run"))
	contains(t, "dry-run did not write", `version = "0.0.1";`, r.ReadFile("flake.nix"))
}

// `format: helm` has fixed keys (version + appVersion), so the target needs no key.
func TestStamp_helmFilter(t *testing.T) {
	t.Parallel()
	const chart = "charts/demo/Chart.yaml"
	r := testrepo.New(t).
		WriteFile(chart, "apiVersion: v2\nname: demo\nversion: 0.0.0\nappVersion: \"0.0.0\"  # trailing comment\n").
		WriteFile(".hanko.yaml", "stamp-targets:\n  - path: charts/demo/Chart.yaml\n    format: helm\n").
		CommitAll("one").Tag("v2.0.0").Commit("two")
	dir := r.Dir()

	contains(t, "dry-run mentions version", "version: 0.0.0 → 2.0.1", ok(t, dir, "stamp", "helm", "--dry-run"))
	contains(t, "dry-run does not write", "version: 0.0.0", r.ReadFile(chart))

	ok(t, dir, "stamp", "helm")
	after := r.ReadFile(chart)
	contains(t, "applies version", "version: 2.0.1", after)
	contains(t, "applies appVersion (quoted)", `appVersion: "2.0.1"`, after)
	contains(t, "preserves trailing comment", "# trailing comment", after)
}

func TestStamp_filterTouchesOnlyMatchingTargets(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).
		WriteFile("VERSION", "0.0.1\n").
		WriteFile("package.json", "{\n  \"name\": \"demo\",\n  \"version\": \"0.0.1\"\n}\n").
		WriteFile(".hanko.yaml", `stamp-targets:
  - path: VERSION
    format: plain
  - path: package.json
    format: json
    key: version
`).
		CommitAll("initial").Tag("v1.0.0").Commit("two")
	dir := r.Dir()

	ok(t, dir, "stamp", "plain")
	eq(t, "plain filter wrote VERSION", "1.0.1", trim(r.ReadFile("VERSION")))
	contains(t, "plain filter left package.json untouched", `"version": "0.0.1"`, r.ReadFile("package.json"))

	// An unknown format must error rather than silently no-op.
	contains(t, "unknown format error names the filter", "xml", fails(t, dir, "stamp", "xml"))
}

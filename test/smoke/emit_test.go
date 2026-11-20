package smoke

import (
	"testing"

	"github.com/dmallubhotla/hanko/internal/testrepo"
)

func TestGoLdflags(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).Commit("one").Tag("v1.0.0").Commit("two")
	out := ok(t, r.Dir(), "version", "go-ldflags")
	contains(t, "has -X main.version", "-X main.version=1.0.1", out)
	contains(t, "has -X main.commit", "-X main.commit=", out)
	contains(t, "has -X main.date", "-X main.date=", out)
	contains(t, "honours --package", "-X example.com/foo.version=1.0.1", ok(t, r.Dir(), "version", "go-ldflags", "--package", "example.com/foo"))
}

func TestDocker(t *testing.T) {
	t.Parallel()
	const img = "ghcr.io/example/demo"
	r := testrepo.New(t).Commit("one").Tag("v1.0.0").Commit("two")
	dir := r.Dir()
	tags := func(t *testing.T, flags ...string) string {
		t.Helper()
		return ok(t, dir, append([]string{"version", "docker", "tags", img}, flags...)...)
	}

	// D-022: 1.0.1 is computed but not released, so no semver refs and no :latest unless edge publishing asks for it.
	t.Run("tags on untagged mainline", func(t *testing.T) {
		branchSha := img + ":main-" + git(r, "rev-parse", "--short", "HEAD")
		eq(t, "branch-sha only", branchSha, tags(t))
		eq(t, "--latest-on-release-tag doesn't fire on mainline", branchSha, tags(t, "--latest-on-release-tag"))
		eq(t, "edge :latest independent of --semver-tags", lines(img+":latest", branchSha), tags(t, "--latest-on-default-branch", "--semver-tags=false"))
	})

	t.Run("tags at a release tag", func(t *testing.T) {
		r.Git("checkout", "-q", "v1.0.0")
		defer r.Git("checkout", "-q", "main")
		eq(t, "full fan-out with release-only :latest",
			lines(img+":1.0.0", img+":1.0", img+":1", img+":latest"),
			tags(t, "--latest-on-release-tag", "--branch-sha-tag=false"))
		eq(t, "no :latest by default",
			lines(img+":1.0.0", img+":1.0", img+":1"),
			tags(t, "--branch-sha-tag=false"))
	})

	t.Run("labels", func(t *testing.T) {
		out := ok(t, dir, "version", "docker", "labels", "--source", "https://example.com/foo", "--title", "demo")
		contains(t, "version label", "--label org.opencontainers.image.version=1.0.1", out)
		contains(t, "revision label", "--label org.opencontainers.image.revision=", out)
		contains(t, "source label", "--label org.opencontainers.image.source=https://example.com/foo", out)
		contains(t, "title label", "--label org.opencontainers.image.title=demo", out)
	})

	// The pre-release counter is commits since v1.0.0 (2), not commits on this branch.
	t.Run("tags on a pre-release", func(t *testing.T) {
		r.Checkout("feature/bar").Commit("three")
		eq(t, "only full + branch-sha",
			lines(img+":1.0.0-feature-bar.2", img+":feature-bar-"+git(r, "rev-parse", "--short", "HEAD")),
			tags(t))
	})
}

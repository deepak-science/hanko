package smoke

import (
	"testing"

	"github.com/dmallubhotla/hanko/internal/testrepo"
)

func TestTag(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).Commit("one").Tag("v1.0.0").Commit("two")
	dir := r.Dir()

	t.Run("happy path", func(t *testing.T) {
		eq(t, "creates v1.0.1", "v1.0.1", ok(t, dir, "tag"))
		eq(t, "tag is annotated", "tag", git(r, "cat-file", "-t", "v1.0.1"))
	})

	t.Run("idempotent rerun", func(t *testing.T) {
		eq(t, "second run echoes existing tag", "v1.0.1", ok(t, dir, "tag"))
	})

	// D-011: `hanko tag` never creates pre-release tags; no escape-hatch flag.
	t.Run("refuses prerelease", func(t *testing.T) {
		r.Checkout("feature/bar").Commit("three")
		out := fails(t, dir, "tag")
		contains(t, "error mentions pre-release", "pre-release", out)
		contains(t, "error suggests merging to main", "merge to main", out)
	})
}

// D-002: a repo bootstrapped with a bare tag keeps getting bare tags.
func TestTag_followsBareTagConvention(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).Commit("one").Tag("1.0.0").Commit("two")
	eq(t, "creates bare 1.0.1", "1.0.1", ok(t, r.Dir(), "tag"))
}

// HEAD is past the latest tag, so the idempotency short-circuit doesn't fire and the dirty check must.
func TestTag_refusesDirty(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).Commit("one").Tag("v1.0.0").Commit("two").WriteFile("c", "dirt\n")
	contains(t, "error mentions dirty", "dirty", fails(t, r.Dir(), "tag"))
}

func TestTag_dryRun(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).Commit("one").Tag("v2.0.0").Commit("two")
	contains(t, "announces would-create", "would create annotated tag", hanko(t, r.Dir(), "tag", "--dry-run").combined)
	eq(t, "no new tag created", "v2.0.0", tagList(r))
}

// D-011 (revised): --initial is the only escape hatch from the pre-release refusal, and only before any semver tag exists.
func TestTag_initial(t *testing.T) {
	t.Parallel()

	t.Run("bootstraps a fresh repo", func(t *testing.T) {
		t.Parallel()
		r := testrepo.New(t).Commit("one")
		eq(t, "creates v0.1.0 verbatim", "v0.1.0", ok(t, r.Dir(), "tag", "--initial", "v0.1.0"))
		eq(t, "initial tag is annotated", "tag", git(r, "cat-file", "-t", "v0.1.0"))
		eq(t, "rerun echoes existing tag", "v0.1.0", ok(t, r.Dir(), "tag", "--initial", "v0.1.0"))
	})

	t.Run("verbatim value preserves bare-tag convention", func(t *testing.T) {
		t.Parallel()
		r := testrepo.New(t).Commit("one")
		eq(t, "creates bare 0.1.0", "0.1.0", ok(t, r.Dir(), "tag", "--initial", "0.1.0"))
	})

	t.Run("refuses if a semver tag exists", func(t *testing.T) {
		t.Parallel()
		r := testrepo.New(t).Commit("one").Tag("v0.1.0").Commit("two")
		contains(t, "error explains why", "--initial only valid when no semver-shaped tag exists", fails(t, r.Dir(), "tag", "--initial", "v1.0.0"))
	})

	t.Run("refuses non-semver values", func(t *testing.T) {
		t.Parallel()
		r := testrepo.New(t).Commit("one")
		contains(t, "error mentions shape", "not a semver-shaped tag", fails(t, r.Dir(), "tag", "--initial", "banana"))
	})
}

// hanko computes v1.0.1 on main; v1.0.1 is squatted on an unreachable side branch.
// git describe is reachable-only so the computation is unaffected, but show-ref is global, so create-time detects the conflict.
func TestTag_refusesConflictingTag(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).Commit("one").Tag("v1.0.0").
		Checkout("side").Commit("side-one").Tag("v1.0.1")
	r.Git("checkout", "-q", "main")
	r.Commit("two")
	contains(t, "error mentions conflict", "does not point at HEAD", fails(t, r.Dir(), "tag"))
}

package smoke

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/dmallubhotla/hanko/internal/testrepo"
)

func TestVersion(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).Commit("one")
	dir := r.Dir()

	t.Run("mainline", func(t *testing.T) {
		eq(t, "no tag", "0.1.0-main.1", ok(t, dir, "version"))
		r.Tag("v1.2.3")
		eq(t, "at tag", "1.2.3", ok(t, dir, "version"))
		r.Commit("two").Commit("three")
		// D-013: mainline past a tag bumps patch by 1 — once — regardless of commit count.
		eq(t, "2 commits past tag on main", "1.2.4", ok(t, dir, "version"))
	})

	t.Run("format gha", func(t *testing.T) {
		got := strings.Split(ok(t, dir, "version", "--format", "gha"), "\n")
		sort.Strings(got)
		want := fmt.Sprintf("branch=main\nfull=1.2.4\nis-prerelease=false\nmajor-minor=1.2\nmajor=1\nminor=2\npatch=4\nshort-sha=%s", git(r, "rev-parse", "--short", "HEAD"))
		eq(t, "--format gha shape", want, strings.Join(got, "\n"))
	})

	t.Run("feature branch", func(t *testing.T) {
		r.Checkout("feature/foo").Commit("four")
		// git describe is reachable-only, so the two mainline commits past v1.2.3 count toward the branch too.
		eq(t, "pre-release on the tag base", "1.2.3-feature-foo.3", ok(t, dir, "version"))
	})
}

func TestVersion_linkedWorktree(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).Commit("one").Tag("v1.0.0").Commit("two")
	r.Git("branch", "feature/wt")
	wt := filepath.Join(t.TempDir(), "linked")
	r.Git("worktree", "add", "-q", wt, "feature/wt")

	eq(t, "main worktree", "1.0.1", ok(t, r.Dir(), "version"))
	contains(t, "linked worktree picks up its own branch", "feature-wt", ok(t, wt, "version"))

	// git keeps per-worktree markers under .git/worktrees/<name>/, which --absolute-git-dir resolves to.
	touch(t, filepath.Join(git(r, "-C", wt, "rev-parse", "--absolute-git-dir"), "MERGE_HEAD"))
	contains(t, "refuses mid-operation in worktree", "merge in progress", fails(t, wt, "version"))
}

// Submodule promise: --repo <parent> reads only the parent's git state.
// A bare nested repo is enough to exercise the boundary; no `git submodule add` needed.
func TestVersion_nestedRepoNotRecursed(t *testing.T) {
	t.Parallel()
	outer := testrepo.New(t).Commit("outer-one").Tag("v1.0.0").Commit("outer-two")
	inner := testrepo.NewAt(t, filepath.Join(outer.Dir(), "sub")).Commit("sub-one").Tag("v9.9.9")

	eq(t, "outer ignores inner tags", "1.0.1", ok(t, outer.Dir(), "version"))
	eq(t, "inner is its own repo", "9.9.9", ok(t, inner.Dir(), "version"))
}

// Detection is path-based, so touching the marker simulates the state without a real merge conflict.
func TestVersion_refusesOperationInProgress(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).Commit("one").Tag("v1.0.0").Commit("two")
	marker := filepath.Join(r.Dir(), ".git", "MERGE_HEAD")
	touch(t, marker)

	out := fails(t, r.Dir(), "version")
	contains(t, "error names the operation", "merge in progress", out)
	contains(t, "error suggests abort", "--abort", out)

	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	eq(t, "works after the operation completes", "1.0.1", ok(t, r.Dir(), "version"))
}

func TestVersion_verbose(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).Commit("one").Tag("v1.0.0").Commit("feat: new shiny")
	res := hanko(t, r.Dir(), "version", "--verbose")
	if res.code != 0 {
		t.Fatalf("exit %d\n%s", res.code, res.combined)
	}
	eq(t, "stdout is just the version", "1.1.0", res.stdout)
	contains(t, "stderr names the strategy", "strategy:        conventional-commits", res.stderr)
	contains(t, "stderr shows the commit subject", "feat: new shiny", res.stderr)
	contains(t, "stderr marks the strongest signal", "← strongest", res.stderr)
	contains(t, "stderr names the direction", "direction:       minor", res.stderr)
}

func TestVersion_bumpOverride(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).Commit("one").Tag("v1.0.0").Commit("chore: no signal here")
	dir := r.Dir()
	eq(t, "no --bump → patch from branch policy", "1.0.1", ok(t, dir, "version"))
	eq(t, "--bump minor", "1.1.0", ok(t, dir, "version", "--bump", "minor"))
	eq(t, "--bump major", "2.0.0", ok(t, dir, "version", "--bump", "major"))
	eq(t, "--bump none", "1.0.0", ok(t, dir, "version", "--bump", "none"))
	contains(t, "bad --bump names valid values", "patch, minor, major, none", fails(t, dir, "version", "--bump", "bananas"))
}

func TestVersion_conventionalCommits(t *testing.T) {
	t.Parallel()
	r := testrepo.New(t).Commit("one").Tag("v1.0.0")
	r.WriteFile(".hanko.yaml", "bump-strategy: conventional-commits\n")
	dir := r.Dir()

	r.Commit("chore: tidy")
	eq(t, "chore alone → fallback patch", "1.0.1", ok(t, dir, "version"))
	r.Commit("feat: shiny new thing")
	eq(t, "feat → minor", "1.1.0", ok(t, dir, "version"))
	r.Commit("feat!: redo api")
	eq(t, "feat! → major", "2.0.0", ok(t, dir, "version"))
}

// Package testrepo is a tiny fluent helper for sketching git states inside
// tests. Exists so packages that touch git (gitinfo, gittag, future stamp)
// can share the same setup vocabulary.
//
// Not part of the public API — internal/* only.
package testrepo

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Builder wraps a temporary git repo with chainable mutations.
type Builder struct {
	t   *testing.T
	dir string
}

// New creates an empty git repo in t.TempDir() with deterministic config.
// The repo starts on `main`. Caller must add at least one commit before
// reading state from it.
func New(t *testing.T) *Builder {
	t.Helper()
	return NewAt(t, t.TempDir())
}

// NewAt is New rooted at dir (created if missing), e.g. a repo nested inside another.
func NewAt(t *testing.T, dir string) *Builder {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	b := &Builder{t: t, dir: dir}
	b.Git("init", "--initial-branch=main", "-q")
	b.Git("config", "user.email", "test@example.invalid")
	b.Git("config", "user.name", "test")
	b.Git("config", "commit.gpgsign", "false")
	b.Git("config", "tag.gpgsign", "false")
	return b
}

// NewBare creates an empty bare repo in t.TempDir() and returns its path, for use as a push remote.
func NewBare(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "origin.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}
	return dir
}

// Dir returns the absolute path to the repo root.
func (b *Builder) Dir() string { return b.dir }

// Git runs `git <args...>` in the repo, fatalling the test on failure.
// Returns the combined output for callers that want to inspect it.
func (b *Builder) Git(args ...string) string {
	b.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = b.dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE=2026-01-01T00:00:00Z",
		"GIT_COMMITTER_DATE=2026-01-01T00:00:00Z",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		b.t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}

// Commit appends `msg` to a file `f` and commits with that message.
func (b *Builder) Commit(msg string) *Builder {
	b.t.Helper()
	f := filepath.Join(b.dir, "f")
	prev, _ := os.ReadFile(f)
	if err := os.WriteFile(f, append(prev, []byte(msg+"\n")...), 0o644); err != nil {
		b.t.Fatal(err)
	}
	b.Git("add", "f")
	b.Git("commit", "-m", msg, "-q")
	return b
}

// CommitAll stages everything in the worktree and commits it with msg.
func (b *Builder) CommitAll(msg string) *Builder {
	b.t.Helper()
	b.Git("add", "-A")
	b.Git("commit", "-m", msg, "-q")
	return b
}

// Tag creates a lightweight tag at HEAD.
func (b *Builder) Tag(name string) *Builder {
	b.t.Helper()
	b.Git("tag", name)
	return b
}

// Checkout creates and switches to a new branch.
func (b *Builder) Checkout(branch string) *Builder {
	b.t.Helper()
	b.Git("checkout", "-q", "-b", branch)
	return b
}

// WriteFile drops a file with the given content into the repo (untracked), creating parent dirs.
func (b *Builder) WriteFile(name, content string) *Builder {
	b.t.Helper()
	p := filepath.Join(b.dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		b.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		b.t.Fatal(err)
	}
	return b
}

// ReadFile returns the contents of a file in the repo.
func (b *Builder) ReadFile(name string) string {
	b.t.Helper()
	data, err := os.ReadFile(filepath.Join(b.dir, name))
	if err != nil {
		b.t.Fatal(err)
	}
	return string(data)
}

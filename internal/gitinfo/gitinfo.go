// Package gitinfo extracts the bits of git state the version engine needs.
//
// Skeleton: shells out to `git`. A future milestone may switch to go-git for
// in-process operation; see ROADMAP.md M1.
package gitinfo

import (
	"os/exec"
	"strings"
)

// Info is a snapshot of the relevant repo state at invocation time.
type Info struct {
	Branch   string
	Sha      string
	ShortSha string
	LatestTag string
	CommitsSinceTag int
	Dirty bool
}

// Read collects the git info for the repo rooted at path.
func Read(path string) (Info, error) {
	info := Info{}

	if out, err := run(path, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		info.Branch = out
	}
	if out, err := run(path, "rev-parse", "HEAD"); err == nil {
		info.Sha = out
	}
	if out, err := run(path, "rev-parse", "--short", "HEAD"); err == nil {
		info.ShortSha = out
	}
	if out, err := run(path, "describe", "--tags", "--abbrev=0"); err == nil {
		info.LatestTag = out
	}
	// Dirty detection: any output from `status --porcelain` means uncommitted changes.
	if out, err := run(path, "status", "--porcelain"); err == nil {
		info.Dirty = out != ""
	}

	return info, nil
}

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

package gitinfo

import (
	"testing"

	"github.com/dmallubhotla/hanko/internal/testrepo"
)

func TestCommitsSince_noTagReturnsAll(t *testing.T) {
	r := testrepo.New(t).Commit("first").Commit("second").Commit("third")
	cs, err := CommitsSince(r.Dir(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 3 {
		t.Fatalf("got %d commits, want 3", len(cs))
	}
	// git log is newest-first.
	if cs[0].Subject != "third" {
		t.Errorf("subjects[0] = %q, want %q", cs[0].Subject, "third")
	}
	if cs[2].Subject != "first" {
		t.Errorf("subjects[2] = %q, want %q", cs[2].Subject, "first")
	}
}

func TestCommitsSince_emptyRangeReturnsNoCommits(t *testing.T) {
	r := testrepo.New(t).Commit("first").Tag("v1.0.0")
	cs, err := CommitsSince(r.Dir(), "v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 0 {
		t.Errorf("got %d commits, want 0", len(cs))
	}
}

func TestCommitsSince_picksUpRangeOnly(t *testing.T) {
	r := testrepo.New(t).
		Commit("first").Tag("v1.0.0").
		Commit("feat: new feature").
		Commit("fix: bug")
	cs, err := CommitsSince(r.Dir(), "v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 2 {
		t.Fatalf("got %d commits, want 2", len(cs))
	}
	subjects := []string{cs[0].Subject, cs[1].Subject}
	wantSet := map[string]bool{"feat: new feature": true, "fix: bug": true}
	for _, s := range subjects {
		if !wantSet[s] {
			t.Errorf("unexpected subject %q", s)
		}
	}
}

func TestCommitsSince_capturesBodySeparately(t *testing.T) {
	r := testrepo.New(t).Commit("first").Tag("v1.0.0")
	r.Git("commit", "--allow-empty", "-q", "-m", "feat: shiny", "-m", "BREAKING CHANGE: the api moved")
	cs, err := CommitsSince(r.Dir(), "v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 1 {
		t.Fatalf("got %d commits, want 1", len(cs))
	}
	if cs[0].Subject != "feat: shiny" {
		t.Errorf("Subject = %q, want %q", cs[0].Subject, "feat: shiny")
	}
	if cs[0].Body == "" || cs[0].Body == "BREAKING CHANGE: the api moved" {
		// Either form is acceptable as long as the body is present.
		// (git uses %b which is the body without the subject.)
	} else if !contains(cs[0].Body, "BREAKING CHANGE: the api moved") {
		t.Errorf("Body = %q, want to contain BREAKING CHANGE line", cs[0].Body)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

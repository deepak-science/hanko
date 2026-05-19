package bump

import (
	"testing"

	"github.com/dmallubhotla/hanko/internal/gitinfo"
)

func TestParse_singleCommitClassifications(t *testing.T) {
	cases := []struct {
		name    string
		subject string
		body    string
		want    Direction
	}{
		{"feat → minor", "feat: add thing", "", Minor},
		{"fix → patch", "fix: bug", "", Patch},
		{"feat with scope", "feat(api): new endpoint", "", Minor},
		{"fix with scope", "fix(parser): off-by-one", "", Patch},
		{"feat! → major (subject !)", "feat!: rip out old api", "", Major},
		{"fix! → major (subject !)", "fix!: change semantics", "", Major},
		{"scoped breaking → major", "feat(api)!: new endpoint", "", Major},
		{"BREAKING CHANGE in body → major", "feat: shiny", "BREAKING CHANGE: the api moved", Major},
		{"BREAKING-CHANGE variant → major", "feat: shiny", "BREAKING-CHANGE: the api moved", Major},
		{"chore → none", "chore: bump deps", "", None},
		{"docs → none", "docs: README updates", "", None},
		{"refactor → none", "refactor: simplify", "", None},
		{"non-conventional subject → none", "Made the thing work", "", None},
		{"empty subject → none", "", "", None},
		{"uppercase type still matched", "FEAT: new", "", Minor},
		{"plain prose with `feat:` inside body line", "feat: shiny", "this is just text, not a footer", Minor},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Parse([]gitinfo.Commit{{Subject: tc.subject, Body: tc.body}})
			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestParse_highestWins(t *testing.T) {
	commits := []gitinfo.Commit{
		{Subject: "fix: small bug"},
		{Subject: "feat: new feature"},
		{Subject: "chore: deps"},
	}
	if got := Parse(commits); got != Minor {
		t.Errorf("got %s, want minor (feat beats fix/chore)", got)
	}

	commits = []gitinfo.Commit{
		{Subject: "fix: a"},
		{Subject: "feat: b"},
		{Subject: "feat!: c"},
	}
	if got := Parse(commits); got != Major {
		t.Errorf("got %s, want major (feat! beats others)", got)
	}
}

func TestParse_emptyListIsNone(t *testing.T) {
	if got := Parse(nil); got != None {
		t.Errorf("got %s, want none", got)
	}
	if got := Parse([]gitinfo.Commit{}); got != None {
		t.Errorf("got %s, want none", got)
	}
}

func TestParse_onlyChoreCommitsIsNone(t *testing.T) {
	commits := []gitinfo.Commit{
		{Subject: "chore: deps"},
		{Subject: "docs: typo"},
		{Subject: "test: add coverage"},
	}
	if got := Parse(commits); got != None {
		t.Errorf("got %s, want none", got)
	}
}

func TestDirection_String(t *testing.T) {
	cases := map[Direction]string{
		Major: "major",
		Minor: "minor",
		Patch: "patch",
		None:  "none",
	}
	for d, want := range cases {
		if got := d.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", d, got, want)
		}
	}
}

// Package bump derives a SemVer bump direction from a list of commits,
// following the Conventional Commits convention.
//
// Hanko owns this so the bump direction is part of the same identity query
// as the rest of the version computation, not delegated to a separate tool.
package bump

import (
	"regexp"
	"strings"

	"github.com/dmallubhotla/hanko/internal/gitinfo"
)

// Direction is a SemVer bump direction. Highest-precedence wins when
// combining multiple commits.
type Direction int

const (
	None Direction = iota
	Patch
	Minor
	Major
)

func (d Direction) String() string {
	switch d {
	case Major:
		return "major"
	case Minor:
		return "minor"
	case Patch:
		return "patch"
	default:
		return "none"
	}
}

// Parse returns "none"/"patch"/"minor"/"major" — see Direction.String — for
// the strongest signal across `commits`. Missing or non-conformant commit
// messages contribute "none".
func Parse(commits []gitinfo.Commit) Direction {
	max := None
	for _, c := range commits {
		d := classify(c)
		if d > max {
			max = d
		}
		// Early-exit: nothing beats Major.
		if max == Major {
			break
		}
	}
	return max
}

// classify maps a single commit to its Direction.
//
// Rules (Conventional Commits):
//   - subject contains `!:` after the type → Major
//   - body line begins with `BREAKING CHANGE:` or `BREAKING-CHANGE:` → Major
//   - subject type is `feat` → Minor
//   - subject type is `fix` → Patch
//   - everything else (`chore:`, `docs:`, `refactor:`, etc.) → None
//
// `<type>(<scope>):` is supported — the scope is treated as opaque.
func classify(c gitinfo.Commit) Direction {
	if breakingInBody(c.Body) || breakingInSubject(c.Subject) {
		return Major
	}
	switch subjectType(c.Subject) {
	case "feat":
		return Minor
	case "fix":
		return Patch
	}
	return None
}

// subjectRE captures the type (group 1) of a Conventional Commits subject.
// `<type>(<scope>)?(!)?: <description>`.
var subjectRE = regexp.MustCompile(`^([a-zA-Z]+)(?:\([^)]*\))?!?:\s`)

func subjectType(subject string) string {
	m := subjectRE.FindStringSubmatch(subject)
	if m == nil {
		return ""
	}
	return strings.ToLower(m[1])
}

// breakingSubjectRE matches `<type>(<scope>)?!:` — a `!` before the colon
// signals a breaking change per the Conventional Commits spec.
var breakingSubjectRE = regexp.MustCompile(`^[a-zA-Z]+(?:\([^)]*\))?!:\s`)

func breakingInSubject(subject string) bool {
	return breakingSubjectRE.MatchString(subject)
}

// breakingInBody returns true if any line of the body begins with the
// `BREAKING CHANGE:` (or `BREAKING-CHANGE:`) footer.
func breakingInBody(body string) bool {
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "BREAKING CHANGE:") ||
			strings.HasPrefix(trim, "BREAKING-CHANGE:") {
			return true
		}
	}
	return false
}

// Package version computes a semantic version from git history.
//
// The skeleton currently returns a placeholder. Full implementation —
// branch-aware bumping, pre-release suffixes, build metadata — lands in
// ROADMAP.md milestone M1.
package version

import "github.com/dmallubhotla/hanko/internal/gitinfo"

// Version is the canonical version output. Field set is loosely modelled on
// GitVersion so existing consumers can be ported with minimal changes.
type Version struct {
	Major         int    `json:"major"`
	Minor         int    `json:"minor"`
	Patch         int    `json:"patch"`
	PreRelease    string `json:"preRelease,omitempty"`
	BuildMetadata string `json:"buildMetadata,omitempty"`

	SemVer     string `json:"semVer"`     // e.g. 1.2.3-alpha.1
	FullSemVer string `json:"fullSemVer"` // e.g. 1.2.3-alpha.1+5.abc1234

	BranchName string `json:"branchName"`
	Sha        string `json:"sha"`
	ShortSha   string `json:"shortSha"`
}

// Compute is the entry point for the version calculation engine.
// Skeleton: returns a hard-coded 0.0.0 based purely on git state.
func Compute(info gitinfo.Info) (Version, error) {
	v := Version{
		Major:      0,
		Minor:      0,
		Patch:      0,
		SemVer:     "0.0.0",
		FullSemVer: "0.0.0",
		BranchName: info.Branch,
		Sha:        info.Sha,
		ShortSha:   info.ShortSha,
	}
	return v, nil
}

// AsEnv flattens the version into HANKO_* environment variables, suitable for
// `eval $(hanko version --format env)` in shell scripts.
func (v Version) AsEnv() map[string]string {
	return map[string]string{
		"HANKO_SEMVER":      v.SemVer,
		"HANKO_FULL_SEMVER": v.FullSemVer,
		"HANKO_BRANCH":      v.BranchName,
		"HANKO_SHA":         v.Sha,
		"HANKO_SHORT_SHA":   v.ShortSha,
	}
}

package cmd

import (
	"reflect"
	"testing"

	"github.com/dmallubhotla/hanko/internal/version"
)

func TestComputeDockerTags(t *testing.T) {
	// Untagged mainline: release-shaped next version, but not released yet.
	mainline := version.Version{
		SemVer: "1.2.3", Major: 1, Minor: 2, Patch: 3,
		BranchName: "main", ShortSha: "abc1234",
	}
	// Pre-first-release mainline: initial-version always carries a pre-release suffix.
	mainlinePre := version.Version{
		SemVer: "0.1.0-main.4", Major: 0, Minor: 1, Patch: 0,
		BranchName: "main", ShortSha: "abc1234", IsPreRelease: true,
	}
	feature := version.Version{
		SemVer: "1.2.3-feature-foo.3", Major: 1, Minor: 2, Patch: 3,
		BranchName: "feature/foo", ShortSha: "abc1234", IsPreRelease: true,
	}
	// D-001 detached-at-tag: BranchName is the sentinel, but AtReleaseTag is set.
	releaseTag := version.Version{
		SemVer: "1.2.3", Major: 1, Minor: 2, Patch: 3,
		BranchName: "detached", ShortSha: "abc1234", AtReleaseTag: true,
	}
	prereleaseTag := version.Version{
		SemVer: "1.2.3-rc.1", Major: 1, Minor: 2, Patch: 3,
		BranchName: "detached", ShortSha: "abc1234", IsPreRelease: true, AtReleaseTag: true,
	}

	cases := []struct {
		name                                  string
		v                                     version.Version
		latest, latestRelease, semver, branch bool
		extras                                []string
		want                                  []string
	}{
		{
			// D-022: the computed next version isn't published, so no :1.2.3 / :1.2 / :1.
			name:   "untagged mainline emits no semver refs",
			v:      mainline,
			semver: true, branch: true,
			want: []string{"main-abc1234"},
		},
		{
			// The taiga-backend invocation: release-only :latest must not fire on mainline.
			name:          "untagged mainline with --latest-on-release-tag: no :latest",
			v:             mainline,
			latestRelease: true, semver: true, branch: true,
			want: []string{"main-abc1234"},
		},
		{
			name:   "untagged mainline with --latest-on-default-branch: edge :latest",
			v:      mainline,
			latest: true, semver: true, branch: true,
			want: []string{"latest", "main-abc1234"},
		},
		{
			// Previously silenced because :latest lived inside the semver block.
			name:   "edge :latest survives --semver-tags=false",
			v:      mainline,
			latest: true, semver: false, branch: true,
			want: []string{"latest", "main-abc1234"},
		},
		{
			name:   "pre-first-release mainline: full prerelease semver, no :latest",
			v:      mainlinePre,
			latest: true, semver: true, branch: true,
			want: []string{"0.1.0-main.4", "main-abc1234"},
		},
		{
			name:   "prerelease feature branch: only full semver + branch-sha",
			v:      feature,
			latest: true, semver: true, branch: true,
			want: []string{"1.2.3-feature-foo.3", "feature-foo-abc1234"},
		},
		{
			name:          "release tag with --latest-on-release-tag fans out with :latest",
			v:             releaseTag,
			latestRelease: true, semver: true, branch: true,
			want: []string{"1.2.3", "1.2", "1", "latest", "detached-abc1234"},
		},
		{
			name:   "release tag without either latest flag fans out without :latest",
			v:      releaseTag,
			semver: true, branch: true,
			want: []string{"1.2.3", "1.2", "1", "detached-abc1234"},
		},
		{
			// --latest-on-default-branch must NOT fire for the detached sentinel.
			name:   "release tag with only --latest-on-default-branch: no :latest",
			v:      releaseTag,
			latest: true, semver: true, branch: true,
			want: []string{"1.2.3", "1.2", "1", "detached-abc1234"},
		},
		{
			name:          "release tag with --semver-tags=false keeps :latest",
			v:             releaseTag,
			latestRelease: true, semver: false, branch: false,
			want: []string{"latest"},
		},
		{
			// Non-prerelease guard still holds: a prerelease tag never moves :latest.
			name:   "prerelease release tag never emits :latest",
			v:      prereleaseTag,
			latest: true, latestRelease: true, semver: true, branch: true,
			want: []string{"1.2.3-rc.1", "detached-abc1234"},
		},
		{
			name:   "semver-tags=false with extras",
			v:      mainline,
			semver: false, branch: false, extras: []string{"edge"},
			want: []string{"edge"},
		},
		{
			name:          "extras appended after computed tags",
			v:             releaseTag,
			latestRelease: true, semver: true, branch: false, extras: []string{"stable", "  ", "extra"},
			want: []string{"1.2.3", "1.2", "1", "latest", "stable", "extra"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeDockerTags(tc.v, tc.latest, tc.latestRelease, tc.semver, tc.branch, tc.extras)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("computeDockerTags() = %v, want %v", got, tc.want)
			}
		})
	}
}

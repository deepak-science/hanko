package cmd

import (
	"reflect"
	"testing"

	"github.com/dmallubhotla/hanko/internal/version"
)

func TestComputeDockerTags(t *testing.T) {
	mainline := version.Version{
		SemVer: "1.2.3", Major: 1, Minor: 2, Patch: 3,
		BranchName: "main", ShortSha: "abc1234",
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
			name:   "mainline non-prerelease fans out with :latest",
			v:      mainline,
			latest: true, semver: true, branch: true,
			want: []string{"1.2.3", "1.2", "1", "latest", "main-abc1234"},
		},
		{
			name:   "mainline without latest flag omits :latest",
			v:      mainline,
			latest: false, semver: true, branch: true,
			want: []string{"1.2.3", "1.2", "1", "main-abc1234"},
		},
		{
			name:   "prerelease feature branch: only full semver + branch-sha",
			v:      feature,
			latest: true, semver: true, branch: true,
			want: []string{"1.2.3-feature-foo.3", "feature-foo-abc1234"},
		},
		{
			// The core issue #1 fix: a tag-push checkout (detached, BranchName
			// sentinel) can opt into :latest via --latest-on-release-tag.
			name:          "release tag with --latest-on-release-tag emits :latest",
			v:             releaseTag,
			latestRelease: true, semver: true, branch: true,
			want: []string{"1.2.3", "1.2", "1", "latest", "detached-abc1234"},
		},
		{
			name:          "release tag without the flag does not emit :latest",
			v:             releaseTag,
			latestRelease: false, semver: true, branch: true,
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
			// Non-prerelease guard still holds: a prerelease tag never moves :latest.
			name:          "prerelease release tag never emits :latest",
			v:             prereleaseTag,
			latestRelease: true, semver: true, branch: true,
			want: []string{"1.2.3-rc.1", "detached-abc1234"},
		},
		{
			// The core issue #2 fix: suppress the semver fan-out, keep branch-sha.
			name:   "semver-tags=false keeps only the branch-sha ref",
			v:      mainline,
			latest: true, semver: false, branch: true,
			want: []string{"main-abc1234"},
		},
		{
			name:   "semver-tags=false with extras",
			v:      mainline,
			semver: false, branch: false, extras: []string{"edge"},
			want: []string{"edge"},
		},
		{
			name:   "extras appended after computed tags",
			v:      mainline,
			latest: true, semver: true, branch: false, extras: []string{"stable", "  ", "extra"},
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

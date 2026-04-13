package main

import "testing"

func TestParseSemver(t *testing.T) {
	cases := []struct {
		in                       string
		maj, min, pat            int
		pre                      string
	}{
		{"v3.0.0", 3, 0, 0, ""},
		{"3.0.0", 3, 0, 0, ""},
		{"v3.1.4", 3, 1, 4, ""},
		{"v3.0.0-beta.2", 3, 0, 0, "beta.2"},
		{"3.0.0-rc.1", 3, 0, 0, "rc.1"},
		{"v3.0.0-alpha", 3, 0, 0, "alpha"},
		// Build metadata is discarded
		{"v3.0.0+build.42", 3, 0, 0, ""},
		{"v3.0.0-beta.2+sha.abcdef", 3, 0, 0, "beta.2"},
		// Missing components default to 0
		{"v2.1", 2, 1, 0, ""},
		{"v2", 2, 0, 0, ""},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			maj, min, pat, pre := parseSemver(c.in)
			if maj != c.maj || min != c.min || pat != c.pat || pre != c.pre {
				t.Errorf("parseSemver(%q) = %d.%d.%d-%q, want %d.%d.%d-%q",
					c.in, maj, min, pat, pre, c.maj, c.min, c.pat, c.pre)
			}
		})
	}
}

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		// Equal
		{"v3.0.0", "v3.0.0", 0},
		{"3.0.0", "v3.0.0", 0},
		{"v3.0.0-beta.2", "v3.0.0-beta.2", 0},

		// Major/minor/patch
		{"v3.0.0", "v2.9.9", 1},
		{"v2.9.9", "v3.0.0", -1},
		{"v3.1.0", "v3.0.9", 1},
		{"v3.0.1", "v3.0.0", 1},

		// Stable > pre-release (SemVer §11.3)
		{"v3.0.0", "v3.0.0-beta.2", 1},
		{"v3.0.0-beta.2", "v3.0.0", -1},
		{"v3.0.0", "v3.0.0-rc.1", 1},

		// Pre-release identifier ordering (§11.4)
		{"v3.0.0-beta.2", "v3.0.0-beta.1", 1},
		{"v3.0.0-beta.1", "v3.0.0-beta.2", -1},
		{"v3.0.0-beta.10", "v3.0.0-beta.2", 1}, // numeric compare, not string
		// numeric < alphanumeric
		{"v3.0.0-1", "v3.0.0-alpha", -1},
		{"v3.0.0-alpha", "v3.0.0-1", 1},
		// Shorter < longer when leading fields match
		{"v3.0.0-beta", "v3.0.0-beta.1", -1},
		{"v3.0.0-beta.1", "v3.0.0-beta", 1},
		// alpha < beta alphanumerically
		{"v3.0.0-alpha", "v3.0.0-beta", -1},
		{"v3.0.0-beta", "v3.0.0-alpha", 1},

	}
	for _, c := range cases {
		t.Run(c.a+"_vs_"+c.b, func(t *testing.T) {
			got := compareSemver(c.a, c.b)
			if got != c.want {
				t.Errorf("compareSemver(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
			}
		})
	}
}

func TestIsNewerVersion(t *testing.T) {
	cases := []struct {
		local, remote string
		want          bool
	}{
		{"v3.0.0", "v3.0.1", true},
		{"v3.0.1", "v3.0.0", false},
		{"v3.0.0", "v3.0.0", false},
		{"v3.0.0-beta.2", "v3.0.0", true},      // stable > pre-release
		{"v3.0.0", "v3.0.0-beta.2", false},     // stable > pre-release
		{"v3.0.0-beta.1", "v3.0.0-beta.2", true},
		{"v3.0.0-beta.2", "v3.0.0-beta.1", false},
	}
	for _, c := range cases {
		t.Run(c.local+"_to_"+c.remote, func(t *testing.T) {
			got := isNewerVersion(c.local, c.remote)
			if got != c.want {
				t.Errorf("isNewerVersion(%q, %q) = %v, want %v", c.local, c.remote, got, c.want)
			}
		})
	}
}

func TestPickLatestPrerelease(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		if got := pickLatestPrerelease(nil); got != nil {
			t.Errorf("want nil, got %+v", got)
		}
	})

	t.Run("no prereleases", func(t *testing.T) {
		rels := []GitHubRelease{
			{TagName: "v3.0.0", Prerelease: false},
			{TagName: "v2.9.0", Prerelease: false},
		}
		if got := pickLatestPrerelease(rels); got != nil {
			t.Errorf("want nil, got %+v", got)
		}
	})

	t.Run("picks highest semver prerelease", func(t *testing.T) {
		rels := []GitHubRelease{
			{TagName: "v3.0.0", Prerelease: false},
			{TagName: "v3.0.0-beta.1", Prerelease: true},
			{TagName: "v3.1.0-beta.2", Prerelease: true},
			{TagName: "v3.0.0-beta.10", Prerelease: true},
			{TagName: "v3.0.0-beta.2", Prerelease: true},
		}
		got := pickLatestPrerelease(rels)
		if got == nil || got.TagName != "v3.1.0-beta.2" {
			t.Errorf("want v3.1.0-beta.2, got %+v", got)
		}
	})

	t.Run("ignores stable entries when picking prerelease", func(t *testing.T) {
		rels := []GitHubRelease{
			{TagName: "v4.0.0", Prerelease: false},
			{TagName: "v3.0.0-beta.2", Prerelease: true},
		}
		got := pickLatestPrerelease(rels)
		if got == nil || got.TagName != "v3.0.0-beta.2" {
			t.Errorf("want v3.0.0-beta.2, got %+v", got)
		}
	})
}

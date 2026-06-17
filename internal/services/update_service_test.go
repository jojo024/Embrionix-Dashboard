package services

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v0.7.0", "v0.6.0", true},
		{"v0.6.1", "v0.6.0", true},
		{"v1.0.0", "v0.9.9", true},
		{"v0.6.0", "v0.6.0", false},
		{"v0.5.0", "v0.6.0", false},
		{"v0.6.0", "v0.6.1", false},
		{"v0.7.0", "dev", false},   // non-semver current never updates
		{"garbage", "v0.6.0", false},
		{"v1.2.3-rc1", "v1.2.2", true}, // pre-release suffix ignored on compare
	}
	for _, c := range cases {
		if got := isNewer(c.latest, c.current); got != c.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestChecksumFor(t *testing.T) {
	contents := "abc123  embrionix-dashboard-linux-amd64\n" +
		"def456 *embrionix-dashboard-windows-amd64.exe\n"

	if got := checksumFor(contents, "embrionix-dashboard-windows-amd64.exe"); got != "def456" {
		t.Errorf("expected def456, got %q", got)
	}
	if got := checksumFor(contents, "embrionix-dashboard-linux-amd64"); got != "abc123" {
		t.Errorf("expected abc123, got %q", got)
	}
	if got := checksumFor(contents, "missing"); got != "" {
		t.Errorf("expected empty for missing file, got %q", got)
	}
}

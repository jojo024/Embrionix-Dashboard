package services

import "testing"

func TestPTPStatusLabel(t *testing.T) {
	cases := []struct {
		code      string
		wantLabel string
		wantLock  bool
	}{
		{"0", "unlocked", false},
		{"1", "coarse lock", false},
		{"3", "locked", true},
		{"0x3", "locked", true},
		{"", "unlocked", false},
		{"7", "code 7", false},
	}
	for _, c := range cases {
		label, locked := ptpStatusLabel(c.code)
		if label != c.wantLabel || locked != c.wantLock {
			t.Errorf("ptpStatusLabel(%q) = (%q, %v), want (%q, %v)",
				c.code, label, locked, c.wantLabel, c.wantLock)
		}
	}
}

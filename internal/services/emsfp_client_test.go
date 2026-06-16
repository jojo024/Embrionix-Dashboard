package services

import "testing"

func TestAsString(t *testing.T) {
	cases := []struct {
		in   interface{}
		want string
	}{
		{"80", "80"},          // string passes through
		{float64(80), "80"},   // JSON number -> string
		{float64(0), "0"},     // numeric VLAN id
		{nil, ""},             // missing key
		{true, ""},            // unsupported type -> empty
	}
	for _, c := range cases {
		if got := asString(c.in); got != c.want {
			t.Errorf("asString(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

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

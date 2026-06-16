package handlers

import "testing"

func TestIsIPv4(t *testing.T) {
	cases := map[string]bool{
		"192.168.1.1":     true,
		"0.0.0.0":         true,
		"255.255.255.255": true,
		"192.168.1.256":   false,
		"":                false,
		"not-an-ip":       false,
		"::1":             false, // IPv6 is not accepted
		"192.168.1":       false,
	}
	for in, want := range cases {
		if got := isIPv4(in); got != want {
			t.Errorf("isIPv4(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestDashIfEmpty(t *testing.T) {
	if dashIfEmpty("") != "-" {
		t.Error("empty string should render as -")
	}
	if dashIfEmpty("x") != "x" {
		t.Error("non-empty string should pass through")
	}
}

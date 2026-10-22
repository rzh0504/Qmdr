package main

import "testing"

func TestIsUpdateAvailable(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"v0.3.1", "0.2.0", true},
		{"0.3.1", "0.3.0", true},
		{"0.3.1", "0.3.1", false},
		{"0.2.0", "0.3.1", false},
		{"0.10.0", "0.9.9", true},
	}
	for _, tt := range tests {
		if got := isUpdateAvailable(tt.latest, tt.current); got != tt.want {
			t.Errorf("isUpdateAvailable(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

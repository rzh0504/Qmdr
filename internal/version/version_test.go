package version

import "testing"

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"v0.2.0", "0.2.0", 0},
		{"0.2", "0.2.0", 0},
		{"0.3.0", "0.2.9", 1},
		{"0.10.0", "0.9.9", 1},
		{"1.0.0", "0.99.99", 1},
		{"v0.2.0-beta.1", "0.2.0", 0},
		{"", "0.2.0", -1},
	}
	for _, tt := range tests {
		if got := Compare(tt.a, tt.b); got != tt.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

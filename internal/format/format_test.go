package format

import "testing"

func TestByteSize(t *testing.T) {
	cases := []struct {
		size int64
		want string
	}{
		{0, ""},
		{-1, ""},
		{512, "512 B"},
		{1 << 10, "1 KB"},
		{3 << 20, "3.0 MB"},
		{3 << 30, "3.0 GB"},
	}
	for _, tc := range cases {
		if got := ByteSize(tc.size); got != tc.want {
			t.Errorf("ByteSize(%d) = %q, want %q", tc.size, got, tc.want)
		}
	}
}

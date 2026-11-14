package qqmusic

import "testing"

func TestAudioMagic(t *testing.T) {
	cases := []struct {
		name string
		head []byte
		want bool
	}{
		{"id3 tag", []byte("ID3\x04\x00"), true},
		{"flac stream", []byte("fLaC\x00\x00"), true},
		{"mpeg frame", []byte{0xFF, 0xFB, 0x90, 0x00}, true},
		{"html page", []byte("<html>"), false},
		{"json body", []byte(`{"code"`), false},
		{"empty", nil, false},
		{"single byte", []byte{0xFF}, false},
	}
	for _, tc := range cases {
		if got := audioMagic(tc.head); got != tc.want {
			t.Fatalf("%s: audioMagic(% x) = %v, want %v", tc.name, tc.head, got, tc.want)
		}
	}
}

package qqmusic

import (
	"testing"
	"time"
)

func TestCredentialExpired(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	cases := []struct {
		name      string
		expiredAt int64
		want      bool
	}{
		{"unknown", 0, false},
		{"future", now.Add(24 * time.Hour).Unix(), false},
		{"within margin", now.Add(5 * time.Minute).Unix(), true},
		{"margin boundary", now.Add(credentialExpiryMargin).Unix(), true},
		{"past", now.Add(-time.Hour).Unix(), true},
		{"milliseconds", now.Add(24 * time.Hour).UnixMilli(), false},
		{"implausible", 123, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			credential := Credential{ExpiredAt: tc.expiredAt}
			if got := credential.Expired(now); got != tc.want {
				t.Fatalf("Expired() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestClientHidesExpiredCredential(t *testing.T) {
	client := &Client{}
	client.setCredential(Credential{MusicID: 42, Musickey: "key", ExpiredAt: time.Now().Add(-time.Minute).Unix()})
	if client.HasCredential() {
		t.Fatal("HasCredential() = true for an expired credential")
	}
	if got := client.UserID(); got != "" {
		t.Fatalf("UserID() = %q, want empty", got)
	}
}

func TestClientUsesFreshCredential(t *testing.T) {
	client := &Client{}
	client.setCredential(Credential{MusicID: 42, Musickey: "key", ExpiredAt: time.Now().Add(time.Hour).Unix()})
	if !client.HasCredential() {
		t.Fatal("HasCredential() = false for a fresh credential")
	}
	if got := client.UserID(); got != "42" {
		t.Fatalf("UserID() = %q, want 42", got)
	}
}

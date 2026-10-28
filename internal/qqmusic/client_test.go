package qqmusic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
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

func TestCredentialNeedsRefresh(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	const lifetime = int64(259200)
	cases := []struct {
		name      string
		created   int64
		expiresIn int64
		want      bool
	}{
		{"unknown", 0, 0, false},
		{"fresh", now.Unix(), lifetime, false},
		{"expired", now.Add(-3 * 24 * time.Hour).Unix(), lifetime, true},
		{"within margin", now.Add(-3*24*time.Hour + 5*time.Minute).Unix(), lifetime, true},
		{"margin boundary", now.Add(-3*24*time.Hour + credentialExpiryMargin).Unix(), lifetime, true},
		{"long lifetime", now.Unix(), 7 * 24 * 3600, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			credential := Credential{MusickeyCreateTime: tc.created, KeyExpiresIn: tc.expiresIn}
			if got := credential.NeedsRefresh(now); got != tc.want {
				t.Fatalf("NeedsRefresh() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMergeCredentialKeepsOldValues(t *testing.T) {
	old := Credential{
		OpenID:             "openid",
		RefreshToken:       "refresh-token",
		ExpiredAt:          123,
		MusicID:            42,
		Musickey:           "old-key",
		RefreshKey:         "refresh-key",
		LoginType:          2,
		MusickeyCreateTime: 1000,
		KeyExpiresIn:       259200,
	}
	fresh := Credential{MusicID: 42, Musickey: "new-key", MusickeyCreateTime: 2000}
	got := mergeCredential(old, fresh)
	if got.OpenID != "openid" || got.RefreshToken != "refresh-token" ||
		got.RefreshKey != "refresh-key" || got.ExpiredAt != 123 ||
		got.MusickeyCreateTime != 2000 || got.KeyExpiresIn != 259200 {
		t.Fatalf("mergeCredential() = %+v", got)
	}

	withoutTime := mergeCredential(old, Credential{MusicID: 42, Musickey: "newer-key"})
	if withoutTime.MusickeyCreateTime <= old.MusickeyCreateTime {
		t.Fatalf("MusickeyCreateTime = %d, want a fresh timestamp", withoutTime.MusickeyCreateTime)
	}
}

func TestCallRefreshesAndRetries(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AppData", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)

	now := time.Now()
	var mu sync.Mutex
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var envelope map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		mu.Lock()
		calls++
		call := calls
		mu.Unlock()
		switch call {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"music.foo.Bar": map[string]any{"code": 1000, "msg": "expired"},
			})
		case 2:
			var refresh struct {
				Param struct {
					RefreshKey   string `json:"refresh_key"`
					RefreshToken string `json:"refresh_token"`
					Musickey     string `json:"musickey"`
					MusicID      int64  `json:"musicid"`
				} `json:"param"`
			}
			if err := json.Unmarshal(envelope["music.login.LoginServer.Login"], &refresh); err != nil {
				t.Errorf("decode refresh: %v", err)
			}
			if refresh.Param.RefreshKey != "old-refresh-key" ||
				refresh.Param.RefreshToken != "old-refresh-token" ||
				refresh.Param.Musickey != "old-key" || refresh.Param.MusicID != 42 {
				t.Errorf("refresh params = %+v", refresh.Param)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"music.login.LoginServer.Login": map[string]any{
					"code": 0,
					"data": map[string]any{
						"musicid":            int64(42),
						"musickey":           "fresh-key",
						"refresh_key":        "fresh-refresh-key",
						"refresh_token":      "fresh-refresh-token",
						"expired_at":         now.Add(20 * 24 * time.Hour).Unix(),
						"musickeyCreateTime": now.Unix(),
						"keyExpiresIn":       int64(259200),
						"login_type":         2,
					},
				},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"music.foo.Bar": map[string]any{"code": 0, "data": map[string]any{"ok": true}},
			})
		}
	}))
	defer server.Close()
	oldEndpoint := apiEndpoint
	apiEndpoint = server.URL
	t.Cleanup(func() { apiEndpoint = oldEndpoint })

	client := &Client{http: newHTTPClient(), QIMEI: fallbackQIMEI}
	client.setCredential(Credential{
		MusicID:            42,
		Musickey:           "old-key",
		RefreshKey:         "old-refresh-key",
		RefreshToken:       "old-refresh-token",
		ExpiredAt:          now.Add(10 * 24 * time.Hour).Unix(),
		MusickeyCreateTime: now.Add(-4 * 24 * time.Hour).Unix(),
		KeyExpiresIn:       259200,
		LoginType:          2,
	})
	var out struct {
		OK bool `json:"ok"`
	}
	if err := client.call(context.Background(), "music.foo", "Bar", map[string]any{}, &out); err != nil {
		t.Fatalf("call: %v", err)
	}
	if !out.OK {
		t.Fatal("retried call did not decode its data")
	}
	mu.Lock()
	gotCalls := calls
	mu.Unlock()
	if gotCalls != 3 {
		t.Fatalf("calls = %d, want 3 (probe, refresh, retry)", gotCalls)
	}
	credential := client.Credential()
	if credential.Musickey != "fresh-key" || credential.RefreshKey != "fresh-refresh-key" {
		t.Fatalf("credential was not refreshed: %+v", credential)
	}
	data, err := os.ReadFile(CredentialPath())
	if err != nil {
		t.Fatalf("read saved credential: %v", err)
	}
	var saved Credential
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("decode saved credential: %v", err)
	}
	if saved.Musickey != "fresh-key" || saved.KeyExpiresIn != 259200 {
		t.Fatalf("saved credential = %+v", saved)
	}
}

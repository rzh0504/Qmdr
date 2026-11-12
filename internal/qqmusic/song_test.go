package qqmusic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestResolveURLFallsBackPerCandidate(t *testing.T) {
	mid := "003aAYrm3GE0X5"
	candidates := fileTypesForQuality("臻品母带")
	available := candidates[3] // FLAC
	var mu sync.Mutex
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var envelope map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		var req struct {
			Param struct {
				Filename []string `json:"filename"`
			} `json:"param"`
		}
		if err := json.Unmarshal(envelope["music.vkey.GetVkey.UrlGetVkey"], &req); err != nil {
			t.Errorf("decode vkey params: %v", err)
			return
		}
		// The real service only resolves the first filename of a request.
		if len(req.Param.Filename) != 1 {
			t.Errorf("request sent %d filenames, want 1", len(req.Param.Filename))
		}
		name := req.Param.Filename[0]
		mu.Lock()
		requests = append(requests, name)
		mu.Unlock()
		purl := ""
		if strings.HasPrefix(name, available.Code) {
			purl = "prefix/song" + available.Ext
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"music.vkey.GetVkey.UrlGetVkey": map[string]any{
				"code": 0,
				"data": map[string]any{
					"midurlinfo": []map[string]any{{
						"filename": name,
						"songmid":  mid,
						"purl":     purl,
					}},
				},
			},
		})
	}))
	defer server.Close()
	oldEndpoint := apiEndpoint
	apiEndpoint = server.URL
	t.Cleanup(func() { apiEndpoint = oldEndpoint })

	client := &Client{http: newHTTPClient(), QIMEI: fallbackQIMEI}
	resolved, err := client.ResolveURL(context.Background(), mid, "臻品母带")
	if err != nil {
		t.Fatal(err)
	}
	want := ResolvedURL{URL: cdnBase + "prefix/song" + available.Ext, Ext: available.Ext, Quality: available.Name}
	if resolved != want {
		t.Fatalf("resolved = %+v, want %+v", resolved, want)
	}
	if len(requests) != 4 {
		t.Fatalf("made %d requests (%v), want 4 up to the FLAC fallback", len(requests), requests)
	}
}

func TestResolveFromEntries(t *testing.T) {
	mid := "003aAYrm3GE0X5"
	master := fileTypesForQuality("臻品母带")
	flac := fileTypesForQuality("FLAC 无损")
	name := func(ft FileType) string { return ft.Code + mid + mid + ft.Ext }

	cases := []struct {
		name       string
		mid        string
		candidates []FileType
		entries    []vkeyEntry
		want       ResolvedURL
		wantOK     bool
	}{
		{
			name:       "picks highest priority available",
			mid:        mid,
			candidates: master,
			entries: []vkeyEntry{
				{Filename: name(master[0])},
				{Filename: name(master[3]), Purl: "prefix/F000.flac"},
				{Filename: name(master[5]), Purl: "prefix/M500.mp3"},
			},
			want:   ResolvedURL{URL: cdnBase + "prefix/F000.flac", Ext: ".flac", Quality: "FLAC"},
			wantOK: true,
		},
		{
			name:       "falls back to the only available quality",
			mid:        mid,
			candidates: master,
			entries: []vkeyEntry{
				{Filename: name(master[5]), Purl: "prefix/M500.mp3"},
			},
			want:   ResolvedURL{URL: cdnBase + "prefix/M500.mp3", Ext: ".mp3", Quality: "MP3 128K"},
			wantOK: true,
		},
		{
			name:       "absolute URL passes through",
			mid:        mid,
			candidates: flac,
			entries: []vkeyEntry{
				{Filename: name(flac[0]), Wifiurl: "https://example.com/song.flac"},
			},
			want:   ResolvedURL{URL: "https://example.com/song.flac", Ext: ".flac", Quality: "FLAC"},
			wantOK: true,
		},
		{
			name:       "positional order when filenames are absent",
			mid:        mid,
			candidates: flac,
			entries: []vkeyEntry{
				{Songmid: mid},
				{Songmid: mid, Purl: "prefix/M800.mp3"},
			},
			want:   ResolvedURL{URL: cdnBase + "prefix/M800.mp3", Ext: ".mp3", Quality: "MP3 320K"},
			wantOK: true,
		},
		{
			name:       "skips entries for another song",
			mid:        mid,
			candidates: flac,
			entries: []vkeyEntry{
				{Songmid: "other", Filename: name(flac[0]), Purl: "prefix/wrong.flac"},
				{Filename: name(flac[2]), Purl: "prefix/right.mp3"},
			},
			want:   ResolvedURL{URL: cdnBase + "prefix/right.mp3", Ext: ".mp3", Quality: "MP3 128K"},
			wantOK: true,
		},
		{
			name:       "no entry serves the song",
			mid:        mid,
			candidates: flac,
			entries: []vkeyEntry{
				{Filename: name(flac[0])},
				{Filename: name(flac[1])},
			},
			wantOK: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := resolveFromEntries(tc.mid, tc.candidates, tc.entries)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if got != tc.want {
				t.Fatalf("resolved = %+v, want %+v", got, tc.want)
			}
		})
	}
}
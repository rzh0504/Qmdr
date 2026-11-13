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

// vkeyServer serves UrlGetVkey and records the requested filenames. respond
// returns the purl the server would hand out for a filename.
func vkeyServer(t *testing.T, mid string, respond func(name string) string) (*httptest.Server, func() []string) {
	t.Helper()
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
		_ = json.NewEncoder(w).Encode(map[string]any{
			"music.vkey.GetVkey.UrlGetVkey": map[string]any{
				"code": 0,
				"data": map[string]any{
					"midurlinfo": []map[string]any{{
						"filename": name,
						"songmid":  mid,
						"purl":     respond(name),
					}},
				},
			},
		})
	}))
	t.Cleanup(server.Close)
	oldEndpoint := apiEndpoint
	apiEndpoint = server.URL
	t.Cleanup(func() { apiEndpoint = oldEndpoint })
	return server, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), requests...)
	}
}

func TestResolveURLFallsBackPerCandidate(t *testing.T) {
	mid := "003aAYrm3GE0X5"
	flac := fileTypeFLAC
	_, requested := vkeyServer(t, mid, func(name string) string {
		if strings.HasPrefix(name, flac.Code) {
			return "prefix/song.flac"
		}
		return ""
	})
	client := &Client{http: newHTTPClient(), QIMEI: fallbackQIMEI}
	resolved, err := client.ResolveURL(context.Background(), mid, "臻品母带", SongFile{})
	if err != nil {
		t.Fatal(err)
	}
	want := ResolvedURL{URL: cdnBase + "prefix/song.flac", Ext: ".flac", Quality: "FLAC"}
	if resolved != want {
		t.Fatalf("resolved = %+v, want %+v", resolved, want)
	}
	if names := requested(); len(names) != 4 {
		t.Fatalf("made %d requests (%v), want 4 up to the FLAC fallback", len(names), names)
	}
}

func TestResolveURLSkipsUnavailableQualities(t *testing.T) {
	mid := "003aAYrm3GE0X5"
	_, requested := vkeyServer(t, mid, func(name string) string {
		return "prefix/song" + fileTypeFLAC.Ext
	})
	client := &Client{http: newHTTPClient(), QIMEI: fallbackQIMEI}
	file := SongFile{
		MediaMid:   "000mediaMid123",
		SizeFLAC:   4096,
		SizeMP3320: 1024,
	}
	resolved, err := client.ResolveURL(context.Background(), mid, "臻品母带", file)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Quality != "FLAC" {
		t.Fatalf("quality = %q, want FLAC", resolved.Quality)
	}
	names := requested()
	if len(names) != 1 {
		t.Fatalf("made %d requests (%v), want only the available FLAC", len(names), names)
	}
	wantName := fileTypeFLAC.Code + file.MediaMid + fileTypeFLAC.Ext
	if names[0] != wantName {
		t.Fatalf("requested %q, want the media mid name %q", names[0], wantName)
	}
}

func TestResolveURLAllQualitiesUnavailable(t *testing.T) {
	mid := "003aAYrm3GE0X5"
	_, requested := vkeyServer(t, mid, func(string) string { return "" })
	client := &Client{http: newHTTPClient(), QIMEI: fallbackQIMEI}
	file := SongFile{SizeAtmos2: 2048}
	if _, err := client.ResolveURL(context.Background(), mid, "FLAC 无损", file); err == nil {
		t.Fatal("ResolveURL succeeded, want an unavailable-quality error")
	}
	if names := requested(); len(names) != 0 {
		t.Fatalf("made requests (%v) for sizes that are all zero", names)
	}
}

func TestResolveEntry(t *testing.T) {
	mid := "003aAYrm3GE0X5"
	cases := []struct {
		name    string
		entries []vkeyEntry
		want    ResolvedURL
		wantOK  bool
	}{
		{
			name: "serves the first entry with a URL",
			entries: []vkeyEntry{
				{Songmid: "other", Purl: "prefix/wrong.flac"},
				{Songmid: mid},
				{Songmid: mid, Purl: "prefix/right.flac"},
			},
			want:   ResolvedURL{URL: cdnBase + "prefix/right.flac", Ext: ".flac", Quality: "FLAC"},
			wantOK: true,
		},
		{
			name:    "absolute URL passes through",
			entries: []vkeyEntry{{Wifiurl: "https://example.com/song.flac"}},
			want:    ResolvedURL{URL: "https://example.com/song.flac", Ext: ".flac", Quality: "FLAC"},
			wantOK:  true,
		},
		{
			name:    "no entry has a URL",
			entries: []vkeyEntry{{Songmid: mid}, {Songmid: "other", Purl: "prefix/x.flac"}},
			wantOK:  false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := resolveEntry(mid, fileTypeFLAC, tc.entries)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if got != tc.want {
				t.Fatalf("resolved = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestExpectedSize(t *testing.T) {
	file := SongFile{
		SizeAtmos2:  2048,
		SizeAtmos51: 4096,
		SizeFLAC:    1024,
		SizeMP3320:  512,
		SizeMP3128:  128,
	}
	if got := ExpectedSize("臻品母带", file); got != 2048 {
		t.Fatalf("臻品母带 size = %d, want the first available advanced quality", got)
	}
	if got := ExpectedSize("FLAC 无损", file); got != 1024 {
		t.Fatalf("FLAC size = %d, want 1024", got)
	}
	if got := ExpectedSize("MP3 320K", file); got != 512 {
		t.Fatalf("MP3 320K size = %d, want 512", got)
	}
	if got := ExpectedSize("FLAC 无损", SongFile{}); got != 0 {
		t.Fatalf("unknown sizes = %d, want 0", got)
	}
}

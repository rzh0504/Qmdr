package qqmusic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// apiStub makes the next API call return data under key.
func apiStub(t *testing.T, key string, data any) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			key: map[string]any{"code": 0, "data": data},
		})
	}))
	t.Cleanup(server.Close)
	oldEndpoint := apiEndpoint
	apiEndpoint = server.URL
	t.Cleanup(func() { apiEndpoint = oldEndpoint })
}

func TestSongMetaReadsDetailBlocks(t *testing.T) {
	apiStub(t, "music.pf_song_detail_svr.get_song_detail_yqq", map[string]any{
		"info": map[string]any{
			"genre":    map[string]any{"content": []map[string]any{{"value": "流行"}}},
			"lan":      map[string]any{"content": []map[string]any{{"value": "国语"}}},
			"pub_time": map[string]any{"content": []map[string]any{{"value": "2022-09-10"}}},
		},
	})
	client := &Client{http: newHTTPClient(), QIMEI: fallbackQIMEI}
	meta, err := client.SongMeta(context.Background(), "mid")
	if err != nil {
		t.Fatal(err)
	}
	want := SongMeta{Language: "国语", Genre: "流行", ReleaseDate: "2022-09-10"}
	if meta != want {
		t.Fatalf("meta = %+v, want %+v", meta, want)
	}
}

func TestSongCreditsPicksWriterRoles(t *testing.T) {
	apiStub(t, "music.sociality.KolWorksTag.SongProducer", map[string]any{
		"Lst": []map[string]any{
			{"Title": "演唱", "Producers": []map[string]any{{"Name": "周深"}}},
			{"Title": "作词", "Producers": []map[string]any{{"Name": "唐恬"}}},
			{"Title": "作曲", "Producers": []map[string]any{{"Name": "钱雷"}, {"Name": "别人"}}},
		},
	})
	client := &Client{http: newHTTPClient(), QIMEI: fallbackQIMEI}
	credits, err := client.SongCredits(context.Background(), "mid")
	if err != nil {
		t.Fatal(err)
	}
	if credits.Lyricist != "唐恬" || credits.Composer != "钱雷 / 别人" {
		t.Fatalf("credits = %+v", credits)
	}
}

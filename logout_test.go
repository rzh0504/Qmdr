package main

import (
	"testing"

	"github.com/rzh0504/qmdr/internal/qqmusic"
)

func TestLogoutResetsAccountState(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AppData", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)

	a := &app{
		api:   &qqmusic.Client{},
		login: loginState{loggedIn: true, userID: "42", loginType: "QQ", nick: "nick"},
		playlists: playlistState{
			owner:    "42",
			items:    []Playlist{{Name: "playlist"}},
			selected: 0,
			songs:    []Song{{Title: "song"}},
		},
	}
	a.logout()

	if a.login.loggedIn || a.login.userID != "未登录" || a.login.loginType != "QQ" {
		t.Fatalf("login after logout: loggedIn=%v userID=%q loginType=%q",
			a.login.loggedIn, a.login.userID, a.login.loginType)
	}
	if a.playlists.owner != "" || len(a.playlists.items) != 0 ||
		len(a.playlists.songs) != 0 || a.playlists.selected != -1 {
		t.Fatalf("playlists after logout: owner=%q items=%d songs=%d selected=%d",
			a.playlists.owner, len(a.playlists.items), len(a.playlists.songs), a.playlists.selected)
	}
	if a.toast != "已退出登录并清除本地凭据" {
		t.Fatalf("toast = %q", a.toast)
	}
}

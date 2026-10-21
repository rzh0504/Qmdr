package main

import (
	"runtime"

	"github.com/egoist/mygo"
)

func (a *app) menu() *mygo.Menu {
	appItems := []*mygo.MenuItem{
		{Label: "关于 Qmdr", Click: func(*mygo.MenuItem, *mygo.Window) { a.showAbout() }},
		{Label: "检查更新…", Click: func(*mygo.MenuItem, *mygo.Window) { a.checkUpdates() }},
		mygo.Separator(),
		{Label: "设置…", Accelerator: "CmdOrCtrl+,", Click: func(*mygo.MenuItem, *mygo.Window) {
			a.openSettings()
		}},
		mygo.Separator(),
		{Role: mygo.RoleQuit, Label: "退出 Qmdr"},
	}
	appMenu := &mygo.MenuItem{Role: mygo.RoleAppMenu, Submenu: appItems}
	if runtime.GOOS != "darwin" {
		appMenu = &mygo.MenuItem{Label: "Qmdr", Submenu: appItems}
	}
	return mygo.NewMenu([]*mygo.MenuItem{
		appMenu,
		{Role: mygo.RoleEditMenu},
		{Label: "视图", Submenu: []*mygo.MenuItem{
			a.menuPage("搜索下载", "CmdOrCtrl+1", "search"),
			a.menuPage("歌单下载", "CmdOrCtrl+2", "playlists"),
			a.menuPage("下载队列", "CmdOrCtrl+3", "queue"),
			a.menuPage("下载历史", "CmdOrCtrl+4", "history"),
			mygo.Separator(),
			{Role: mygo.RoleToggleFullScreen},
		}},
		{Label: "窗口", Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleMinimize},
			{Role: mygo.RoleZoom},
			mygo.Separator(),
			{Role: mygo.RoleClose},
		}},
		{Label: "帮助", Submenu: []*mygo.MenuItem{
			{Label: "项目主页", Click: func(*mygo.MenuItem, *mygo.Window) { a.openProjectPage() }},
		}},
	})
}

func (a *app) menuPage(label, accelerator, page string) *mygo.MenuItem {
	return &mygo.MenuItem{
		Label:       label,
		Accelerator: accelerator,
		Click: func(*mygo.MenuItem, *mygo.Window) {
			a.update(func() { a.setPage(page) })
		},
	}
}

func (a *app) openSettings() {
	a.update(func() { a.view = viewSettings })
}

func (a *app) openURL(url string) {
	a.update(func() {
		if err := mygo.Shell.OpenExternal(url); err != nil {
			a.toast = "打开链接失败：" + err.Error()
		}
	})
}

func (a *app) openProjectPage() {
	a.openURL(repoURL)
}

func (a *app) showAbout() {
	win := a.win
	go func() {
		result, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:        win,
			Type:          mygo.MessageInfo,
			Title:         "关于 Qmdr",
			Message:       "Qmdr " + appVersion,
			Detail:        "QQ 音乐下载器\nGPL-3.0 · MyGo 原生 UI",
			Buttons:       []string{"项目主页", "关闭"},
			DefaultButton: 0,
			CancelButton:  1,
		})
		if err == nil && result.Button == 0 {
			a.openProjectPage()
		}
	}()
}

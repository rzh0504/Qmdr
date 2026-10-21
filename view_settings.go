package main

import (
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/qqmusic"
)

func (a *app) settingsView(c *ui.Context, col colors) {
	ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
		form := ui.Column(c).FillWidth().MaxWidth(720).Gap(22)
		form.Children(func() {
			a.settingsGroup(c, col, "账号", func() {
				a.accountSettings(c, col)
			})
			a.appearanceSettings(c, col)
			a.settingsGroup(c, col, "下载", func() {
				a.downloadSettings(c, col)
			})
			a.settingsGroup(c, col, "关于", func() {
				a.aboutSettings(c, col)
			})
		})
	})
}

func (a *app) settingsGroup(c *ui.Context, col colors, title string, fn func()) {
	ui.Column(c).FillWidth().Gap(8).Children(func() {
		groupTitle(c, col, title)
		group := ui.Column(c).FillWidth().Padding(2, 16)
		smoothFlat(group, col, 12, col.chip)
		group.Children(fn)
	})
}

func (a *app) settingRow(c *ui.Context, col colors, label, detail string, first bool, control func()) {
	if !first {
		divider(c, col)
	}
	ui.Row(c).FillWidth().MinHeight(46).Padding(11, 0).Gap(16).
		AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
			ui.Text(c, label).FontSize(13).TextColor(col.text)
			if detail != "" {
				ui.Text(c, detail).FontSize(12).TextColor(col.muted).LineHeight(1.35)
			}
		})
		ui.Row(c).Shrink(0).Gap(8).AlignItems(ui.Center).Children(control)
	})
}

func (a *app) textField(c *ui.Context, col colors, value *string, placeholder string, width, height float32, onChange, onSubmit func()) ui.Element {
	frame := ui.Row(c).Width(width).Height(height).Padding(0, 10).AlignItems(ui.Center)
	var input ui.Element
	frame.Children(func() {
		input = ui.TextInputBase(c, value).
			Placeholder(placeholder).
			Label(placeholder).
			FontSize(13).Grow(1).MinWidth(0).Height(height - 4).
			FocusRing(false)
	})
	if input.Changed() && onChange != nil {
		onChange()
	}
	if input.Submitted() && onSubmit != nil {
		onSubmit()
	}
	glow := frame.Animate("focus", b2f(input.Focused()), 120*time.Millisecond)
	smoothField(frame, col, 7, col.control, col.fieldStroke, col.accent, glow)
	return frame
}

func (a *app) accountSettings(c *ui.Context, col colors) {
	ui.Row(c).FillWidth().MinHeight(58).Padding(11, 0).Gap(10).
		AlignItems(ui.Center).Children(func() {
		a.avatar(c, col, 34)
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, "Qmdr 账号").FontSize(13).FontWeight(400).TextColor(col.text)
			ui.Textf(c, "%s · %s 登录", a.userID, a.loginType).
				FontSize(12).TextColor(col.muted)
		})
		status := "未登录"
		dot := col.faint
		if a.loggedIn {
			status = "已登录"
			dot = col.green
		}
		ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(6, 6).Radius(3).Background(dot)
			ui.Text(c, status).FontSize(12).TextColor(col.muted)
		})
	})
	a.settingRow(c, col, "登录方式", "", false, func() {
		primaryButton(c, col, iconLogin, "QQ 登录", func() {
			a.openQR("QQ")
		})
		secondaryButton(c, col, iconLogin, "微信登录", func() {
			a.openQR("微信")
		})
	})
	a.settingRow(c, col, "本地凭证", "重新读取或导出保存的登录凭证", false, func() {
		ghostButton(c, col, iconRefresh, "重新加载", func() {
			a.reloadCredential()
		})
		ghostButton(c, col, iconExternal, "导出 JSON", func() {
			a.exportCredential()
		})
	})
}

func (a *app) appearanceSettings(c *ui.Context, col colors) {
	a.settingsGroup(c, col, "外观", func() {
		a.settingRow(c, col, "主题", "", true, func() {
			stringSegmented(c, col, &a.appearance,
				[]string{"light", "dark", "system"},
				[]string{"亮色", "暗色", "跟随系统"},
				a.applyAppearance)
		})
	})
	ui.Column(c).FillWidth().Gap(8).Children(func() {
		groupTitle(c, col, "背景")
		ui.Row(c).Gap(12).Wrap().Children(func() {
			for _, w := range backgroundWashes {
				a.backgroundTile(c, col, w)
			}
		})
	})
}

func (a *app) applyAppearance() {
	a.saveSettings()
	mygo.Theme.SetSource(themeSource(a.appearance))
	a.syncWindowBackground()
	a.refresh()
}

func (a *app) backgroundTile(c *ui.Context, col colors, w wash) {
	selected := a.background == w.id
	tile := ui.ButtonBase(c).Column().Width(104).Gap(8).AlignItems(ui.Stretch).
		Cursor(ui.CursorPointer).Label(w.name)
	if tile.Clicked() {
		a.background = w.id
		a.saveSettings()
		a.syncWindowBackground()
		a.refresh()
	}
	tile.Children(func() {
		swatch := ui.Box(c).Size(104, 68)
		base, glow := w.gradients(col.dark)
		smoothGradientTile(swatch, col, 12, base, glow)
		swatch.Children(func() {
			ui.Box(c).Absolute().Top(12).Left(26).Right(8).Bottom(8).Radius(7).
				Background(col.card)
			ui.Box(c).Absolute().Top(20).Left(8).Width(12).Height(4).Radius(2).
				Background(col.faint.Alpha(0.5))
			if selected {
				ui.Box(c).Absolute().Top(6).Right(6).Size(18, 18).Radius(9).Center().
					Background(col.accent).Children(func() {
					ui.Icon(c, iconCheck).Size(10, 10).TextColor(col.accentText)
				})
			}
		})
		label := ui.Text(c, w.name).FontSize(13).TextColor(col.muted).
			TextAlign(ui.Center)
		if selected {
			label.TextColor(col.text).FontWeight(600)
		}
	})
}

func (a *app) downloadSettings(c *ui.Context, col colors) {
	a.settingRow(c, col, "下载目录", "", true, func() {
		a.textField(c, col, &a.downloadDir, "选择下载目录", 300, 30, func() {
			a.saveSettings()
		}, nil)
		secondaryButton(c, col, iconFolder, "选择…", func() {
			a.pickDir()
		})
	})
	a.settingRow(c, col, "默认音质", "", false, func() {
		selectControl(c, col, &a.quality, qualityOptions, 160, col.control, func() {
			a.saveSettings()
		})
	})
	a.settingRow(c, col, "封面尺寸", "", false, func() {
		selectControl(c, col, &a.coverSize, coverSizeOptions, 100, col.control, func() {
			a.saveSettings()
		})
	})
	a.settingRow(c, col, "下载并发", "同时下载的歌曲数量", false, func() {
		stepperControl(c, col, &a.concurrency, 1, 8, 1, func() {
			a.saveSettings()
		})
	})
	a.settingRow(c, col, "覆盖已存在的文件", "关闭时，同名文件会自动跳过", false, func() {
		toggle := toggleSwitch(c, col, &a.overwrite)
		if toggle.Changed() {
			a.saveSettings()
		}
	})
}

func (a *app) aboutSettings(c *ui.Context, col colors) {
	ui.Row(c).FillWidth().MinHeight(56).Padding(11, 0).AlignItems(ui.Center).Children(func() {
		a.logoMark(c, col, 24)
		ui.Column(c).Gap(2).Padding(0, 0, 0, 10).Children(func() {
			ui.Text(c, "Qmdr").FontSize(13).FontWeight(400).TextColor(col.text)
			ui.Textf(c, "版本 %s · GPL-3.0 · MyGo 原生 UI", appVersion).
				FontSize(12).TextColor(col.muted)
		})
		ui.Spacer(c)
		if a.updating {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c).Size(13, 13)
				ui.Text(c, "检查中…").FontSize(12).TextColor(col.muted)
			})
			return
		}
		secondaryButton(c, col, iconRefresh, "检查更新", a.checkUpdates)
	})
}

func (a *app) pickDir() {
	win := a.win
	if win == nil {
		return
	}
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:    win,
			Title:     "选择下载目录",
			Directory: true,
		})
		a.update(func() {
			if err != nil {
				a.toast = "打开目录选择器失败：" + err.Error()
				return
			}
			if len(paths) == 0 {
				a.toast = "已取消目录选择"
				return
			}
			a.downloadDir = paths[0]
			a.saveSettings()
			a.appendLog("下载目录已更新："+a.downloadDir, "info")
			a.toast = "下载目录已更新"
		})
	}()
}

func (a *app) reloadCredential() {
	ok := a.api.ReloadCredential()
	a.loggedIn = ok
	a.userID = a.api.UserID()
	if a.userID == "" {
		a.userID = "未登录"
	}
	if a.api.Credential().LoginType == 1 {
		a.loginType = "微信"
	} else {
		a.loginType = "QQ"
	}
	a.playlistReq++
	a.playlistSongsID++
	a.loadingPlaylists = false
	a.refreshingPlaylists = false
	a.loadingSongs = false
	a.playlistOwner = a.api.UserID()
	a.musicIDOpen = false
	a.musicIDInput = ""
	a.musicIDErr = ""
	a.playlistErr = ""
	a.playlists = nil
	a.selectedPlaylist = -1
	a.playlistSongs = nil
	a.restorePlaylistCache()
	if ok {
		a.setToast("已重新加载本地凭证")
	} else {
		a.setToast("未找到有效凭证")
	}
}

func (a *app) exportCredential() {
	path := qqmusic.CredentialPath()
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		a.setToast("没有可导出的本地凭证")
		return
	}
	win := a.win
	if win == nil {
		return
	}
	go func() {
		selected, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent:      win,
			Title:       "导出凭证 JSON",
			DefaultPath: "qqmusic_cred.json",
			Filters:     []mygo.FileFilter{{Name: "JSON", Extensions: []string{"json"}}},
		})
		if err != nil {
			a.update(func() { a.toast = "导出凭证失败：" + err.Error() })
			return
		}
		if selected == "" {
			return
		}
		if !strings.HasSuffix(strings.ToLower(selected), ".json") {
			selected += ".json"
		}
		if err := os.WriteFile(selected, data, 0o600); err != nil {
			a.update(func() { a.toast = "写入凭证失败：" + err.Error() })
			return
		}
		a.update(func() { a.toast = "凭证已导出" })
	}()
}

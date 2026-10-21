package main

import (
	"os"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/qqmusic"
)

func (a *app) settingsView(c *ui.Context, col colors) {
	pageHeader(c, col, "设置", func() {
		secondaryButton(c, col, iconFolder, "打开下载目录", func() {
			a.openDownloadDir()
		})
	})

	form := ui.Column(c).FillWidth().MaxWidth(760).Gap(16)
	form.Children(func() {
		a.settingsSection(c, col, "账号", func() {
			a.accountSettings(c, col)
		})
		divider(c, col)
		a.settingsSection(c, col, "下载", func() {
			a.downloadSettings(c, col)
		})
		divider(c, col)
		a.settingsSection(c, col, "高级", func() {
			a.advancedSettings(c, col)
		})
		divider(c, col)
		a.aboutSettings(c, col)
	})
}

func (a *app) settingsSection(c *ui.Context, col colors, title string, fn func()) {
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, title).FontSize(11).Bold().TextColor(col.muted)
		fn()
	})
}

func (a *app) settingsRow(c *ui.Context, col colors, label string, control func()) {
	ui.Row(c).FillWidth().MinHeight(32).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).Width(130).FontSize(11).TextColor(col.muted)
		ui.Row(c).Grow(1).AlignItems(ui.Center).Children(control)
	})
}

func (a *app) accountSettings(c *ui.Context, col colors) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
		a.avatar(c, col, 34)
		ui.Box(c).Width(8)
		ui.Column(c).Gap(1).Grow(1).Children(func() {
			ui.Text(c, "Qmdr 账号").FontSize(12).Bold().TextColor(col.text)
			ui.Textf(c, "%s · %s 登录", a.userID, a.loginType).
				FontSize(10).TextColor(col.muted)
		})
		status := "未登录"
		dot := col.faint
		if a.loggedIn {
			status = "已登录"
			dot = col.green
		}
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(6, 6).Radius(3).Background(dot)
			ui.Text(c, status).FontSize(10).TextColor(col.muted)
		})
	})

	ui.Row(c).Gap(6).AlignItems(ui.Center).Wrap().Children(func() {
		primaryButton(c, col, iconLogin, "QQ 登录", func() {
			a.openQR("QQ")
		})
		secondaryButton(c, col, iconLogin, "微信登录", func() {
			a.openQR("微信")
		})
		ghostButton(c, col, iconRefresh, "重新加载凭证", func() {
			a.reloadCredential()
		})
		ghostButton(c, col, iconExternal, "导出 JSON", func() {
			a.exportCredential()
		})
	})
}

func (a *app) downloadSettings(c *ui.Context, col colors) {
	a.settingsRow(c, col, "下载目录", func() {
		input := ui.TextInput(c, &a.downloadDir).
			Placeholder("选择下载目录").
			Label("下载目录").
			Grow(1).Height(30).
			Background(col.cardAlt).Border(1, col.border).Radius(6)
		if input.Changed() {
			a.saveSettings()
		}
		secondaryButton(c, col, iconFolder, "选择…", func() {
			a.pickDir()
		})
	})
	a.settingsRow(c, col, "默认音质", func() {
		selectInput := ui.Select(c, &a.quality, qualityOptions).Width(180)
		if selectInput.Changed() {
			a.saveSettings()
		}
	})
	a.settingsRow(c, col, "封面尺寸", func() {
		selectInput := ui.Select(c, &a.coverSize, coverSizeOptions).Width(120)
		if selectInput.Changed() {
			a.saveSettings()
		}
	})
	a.settingsRow(c, col, "覆盖文件", func() {
		checkbox := ui.Checkbox(c, &a.overwrite, "覆盖已存在的文件")
		if checkbox.Changed() {
			a.saveSettings()
		}
	})
}

func (a *app) advancedSettings(c *ui.Context, col colors) {
	a.settingsRow(c, col, "外部凭证 API", func() {
		ui.TextInput(c, &a.externalAPI).
			Placeholder("https://example.com").
			Label("外部凭证 API").
			Grow(1).Height(30).
			Background(col.cardAlt).Border(1, col.border).Radius(6)
		secondaryButton(c, col, iconRefresh, "重新加载", func() {
			a.reloadExternalAPI()
		})
	})
}

func (a *app) aboutSettings(c *ui.Context, col colors) {
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, "关于").FontSize(11).Bold().TextColor(col.muted)
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			a.logoMark(c, col, 22)
			ui.Box(c).Width(8)
			ui.Text(c, "Qmdr").FontSize(12).Bold().TextColor(col.text)
			ui.Box(c).Width(8)
			ui.Text(c, "版本 0.2.0 · GPL-3.0").FontSize(10).TextColor(col.faint)
			ui.Spacer(c)
			ui.Text(c, "MyGo 原生 UI").FontSize(10).TextColor(col.faint)
		})
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
	if a.api.Credential.LoginType == 1 {
		a.loginType = "微信"
	} else {
		a.loginType = "QQ"
	}
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

func (a *app) reloadExternalAPI() {
	if strings.TrimSpace(a.externalAPI) == "" {
		a.setToast("请先输入外部凭证 API 地址")
		return
	}
	a.setToast("已重新加载外部凭证（UI 演示）")
}

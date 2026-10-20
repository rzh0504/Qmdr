package main

import (
	"os"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/qqmusic"
)

func (a *app) settingsView(c *ui.Context, col colors) {
	pageHeader(c, col, "设置", "管理账号凭证、下载位置与音质偏好", func() {
		secondaryButton(c, col, iconFolder, "打开下载目录", func() {
			a.openDownloadDir()
		})
	})

	ui.Row(c).Gap(16).AlignItems(ui.Start).Children(func() {
		a.accountCard(c, col)
		a.downloadSettingsCard(c, col)
	})

	a.advancedCard(c, col)
	a.aboutCard(c, col)
}

func (a *app) accountCard(c *ui.Context, col colors) {
	box := ui.Column(c).Width(350).Padding(16).Gap(12).Radius(10).
		Background(col.card).Border(1, col.border).
		Shadow(0, 1, 3, 0, col.shadow)
	box.Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			a.avatar(c, col, 40)
			ui.Box(c).Width(10)
			ui.Column(c).Gap(2).Grow(1).Children(func() {
				ui.Text(c, "Qmdr 用户").FontSize(14).Bold().TextColor(col.text)
				ui.Textf(c, "%s · %s 登录", a.userID, a.loginType).
					FontSize(11).TextColor(col.muted)
			})
			ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
				status := "未登录"
				dot := col.faint
				if a.loggedIn {
					status = "已登录"
					dot = col.green
				}
				ui.Box(c).Size(6, 6).Radius(3).Background(dot)
				ui.Text(c, status).FontSize(10).TextColor(col.muted)
			})
		})

		divider(c, col)

		ui.Text(c, "扫码登录后可下载 VIP / 无损音质，并读取你的 QQ 音乐歌单。").
			FontSize(11).TextColor(col.muted).MaxLines(2)

		ui.Row(c).Gap(6).Children(func() {
			primaryButton(c, col, iconLogin, "QQ 扫码登录", func() {
				a.openQR("QQ")
			})
			secondaryButton(c, col, iconLogin, "微信登录", func() {
				a.openQR("微信")
			})
		})

		ui.Row(c).Gap(2).Children(func() {
			ghostButton(c, col, iconRefresh, "重新加载凭证", func() {
				a.reloadCredential()
			})
			ghostButton(c, col, iconExternal, "导出 JSON", func() {
				a.exportCredential()
			})
		})

		ui.Row(c).Gap(6).AlignItems(ui.Start).Children(func() {
			ui.Icon(c, iconKey).FontSize(12).TextColor(col.faint)
			ui.Text(c, "凭证将保存到系统钥匙串，而不是明文文件。").
				FontSize(10).TextColor(col.faint).Grow(1).MaxLines(2)
		})
	})
}

func (a *app) downloadSettingsCard(c *ui.Context, col colors) {
	box := ui.Column(c).Grow(1).Padding(16).Gap(14).Radius(10).
		Background(col.card).Border(1, col.border).
		Shadow(0, 1, 3, 0, col.shadow)
	box.Children(func() {
		sectionHeader(c, col, "下载设置", "应用到后续加入队列的任务", nil)

		ui.Column(c).Gap(5).Children(func() {
			ui.Text(c, "下载目录").FontSize(11).TextColor(col.muted)
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.TextInput(c, &a.downloadDir).
					Placeholder("选择下载目录").
					Label("下载目录").
					Grow(1).Height(32).
					Background(col.cardAlt).Border(1, col.border).Radius(7)
				secondaryButton(c, col, iconFolder, "选择目录…", func() {
					a.pickDir()
				})
			})
		})

		divider(c, col)

		ui.Row(c).Gap(16).AlignItems(ui.Start).Wrap().Children(func() {
			ui.Column(c).Gap(5).Width(180).Children(func() {
				ui.Text(c, "默认音质").FontSize(11).TextColor(col.muted)
				ui.Select(c, &a.quality, qualityOptions).Width(180)
			})
			ui.Column(c).Gap(5).Width(130).Children(func() {
				ui.Text(c, "封面尺寸").FontSize(11).TextColor(col.muted)
				ui.Select(c, &a.coverSize, coverSizeOptions).Width(130)
			})
			ui.Column(c).Gap(8).Padding(22, 0, 0, 0).Children(func() {
				ui.Checkbox(c, &a.overwrite, "覆盖已存在文件")
			})
		})

		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "所选音质不可用时自动降级。").
				FontSize(10).TextColor(col.faint).Grow(1)
			ghostButton(c, col, iconFolder, "打开目录", func() {
				a.openDownloadDir()
			})
		})
	})
}

func (a *app) advancedCard(c *ui.Context, col colors) {
	card(c, col, func() {
		sectionHeader(c, col, "高级", "外部凭证服务", nil)
		ui.Row(c).Gap(8).AlignItems(ui.End).Children(func() {
			ui.Column(c).Gap(5).Grow(1).Children(func() {
				ui.Text(c, "外部凭证 API").FontSize(11).TextColor(col.muted)
				ui.TextInput(c, &a.externalAPI).
					Placeholder("https://example.com").
					Label("外部凭证 API").
					Height(32).
					Background(col.cardAlt).Border(1, col.border).Radius(7)
			})
			secondaryButton(c, col, iconRefresh, "重新加载", func() {
				a.reloadExternalAPI()
			})
		})
		ui.Text(c, "留空时使用本地保存的凭证。").
			FontSize(10).TextColor(col.faint)
	})
}

func (a *app) aboutCard(c *ui.Context, col colors) {
	card(c, col, func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			a.logoMark(c, col, 28)
			ui.Box(c).Width(8)
			ui.Column(c).Gap(1).Grow(1).Children(func() {
				ui.Text(c, "Qmdr · UI 预览").FontSize(13).Bold().TextColor(col.text)
				ui.Text(c, "基于 MyGo 原生 UI 构建，网络层将在后续接入。").
					FontSize(10).TextColor(col.muted)
			})
			pill(c, "v0.2.0 · GPL-3.0", col.muted, col.cardAlt)
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
			a.appendLog("下载目录已更新："+a.downloadDir, "info")
			a.toast = "下载目录已更新（UI 演示）"
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

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/version"
)

// appVersion is what local `go build` executables report; mygo build 链接
// mygo.json 里的版本，检查更新的回退逻辑与日志使用它。
var appVersion = "0.4.0"

const (
	repoURL           = "https://github.com/rzh0504/Qmdr"
	latestReleasePage = repoURL + "/releases/latest"
	latestReleaseAPI  = "https://api.github.com/repos/rzh0504/Qmdr/releases/latest"
)

type releaseInfo struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
}

// checkUpdates 由菜单与设置页调用。mygo 构建走应用内更新（含增量）；本地
// go build 等未启用更新的构建退回打开发布页。
func (a *app) checkUpdates() {
	if !mygo.Updater.Enabled() {
		a.checkUpdatesOnWeb()
		return
	}
	if a.updating {
		a.setToast("正在检查更新…")
		return
	}
	a.updating = true
	a.refresh()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		update, err := mygo.Updater.Check(ctx)
		a.update(func() {
			a.updating = false
			switch {
			case err != nil:
				a.toast = "检查更新失败：" + friendlyError(err)
			case update == nil:
				a.toast = "已是最新版本 v" + appVersion
			default:
				a.updateReady = update
				a.updateInfo = updatePrompt{open: true, version: update.Version, notes: update.Notes}
			}
			a.refresh()
		})
	}()
}

// checkUpdatesOnWeb 是未启用应用内更新的构建的回退：比较 GitHub 最新发布
// 的标签，比当前新就打开发布页。
func (a *app) checkUpdatesOnWeb() {
	if a.updating {
		a.setToast("正在检查更新…")
		return
	}
	a.updating = true
	a.refresh()
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("update check panic: %v\n%s", recovered, debug.Stack())
				a.update(func() {
					a.updating = false
					a.toast = "检查更新时出现异常，请稍后重试"
					a.refresh()
				})
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		release, err := fetchLatestRelease(ctx)
		a.update(func() {
			a.updating = false
			switch {
			case err != nil:
				a.toast = "检查更新失败：" + err.Error()
			case !isUpdateAvailable(strings.TrimPrefix(release.TagName, "v"), appVersion):
				a.toast = "已是最新版本 v" + appVersion
			default:
				detail := "前往 GitHub 下载新版本。"
				if release.Name != "" && !strings.EqualFold(release.Name, release.TagName) {
					detail = release.Name + "\n\n" + detail
				}
				releaseURL := release.HTMLURL
				if releaseURL == "" {
					releaseURL = repoURL + "/releases/latest"
				}
				a.updateInfo = updatePrompt{
					open:    true,
					version: strings.TrimPrefix(release.TagName, "v"),
					detail:  detail,
					url:     releaseURL,
				}
			}
			a.refresh()
		})
	}()
}

// installUpdate 下载并应用检查更新时找到的新版本。delta 更新只下载变化的
// 部分，失败自动回退整包；完成后由用户重启生效。
func (a *app) installUpdate() {
	update := a.updateReady
	if update == nil || a.updateInstalling || a.updateInstalled {
		return
	}
	a.updateInstalling = true
	a.updateProgress = 0
	a.refresh()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		err := update.Install(ctx, func(downloaded, total int64) {
			progress := 0.0
			if total > 0 {
				progress = min(1, float64(downloaded)/float64(total))
			}
			a.update(func() {
				a.updateProgress = progress
				a.refresh()
			})
		})
		a.update(func() {
			a.updateInstalling = false
			if err != nil {
				a.toast = "更新安装失败：" + friendlyError(err)
				a.refresh()
				return
			}
			a.updateInstalled = true
			a.refresh()
		})
	}()
}

func isUpdateAvailable(latest, current string) bool {
	return version.Compare(latest, current) > 0
}

func (a *app) updateModal(c *ui.Context, col colors) {
	a.modal(c, col, &a.updateInfo.open, "发现新版本", 440, func(close func()) {
		ui.Row(c).Gap(10).AlignItems(ui.Start).Children(func() {
			ui.Icon(c, iconUpdate).Size(20, 20).TextColor(col.accent)
			ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
				ui.Textf(c, "Qmdr v%s 已发布", a.updateInfo.version).FontSize(14).TextColor(col.text)
				if detail := a.updateInfo.detail; detail != "" {
					ui.Text(c, detail).FontSize(12).TextColor(col.muted).LineHeight(1.45)
				}
			})
		})
		if a.updateInfo.notes != "" {
			notes := ui.Scroll(c).FillWidth().Height(170)
			notes.Children(func() {
				ui.Text(c, a.updateInfo.notes).FontSize(12).TextColor(col.muted).
					LineHeight(1.5).FillWidth()
			})
		}
		switch {
		case a.updateInstalled:
			track := ui.Box(c).Height(4).FillWidth().Radius(2).Background(col.chip)
			track.Children(func() {
				ui.Box(c).Height(4).WidthPercent(100).Radius(2).Background(col.green)
			})
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "更新完成，重启应用生效").FontSize(12).TextColor(col.muted).Grow(1)
				primaryButton(c, col, iconPlay, "重启应用", func() { mygo.App.Relaunch() })
			})
		case a.updateInstalling:
			track := ui.Box(c).Height(4).FillWidth().Radius(2).Background(col.chip)
			track.Children(func() {
				if a.updateProgress > 0 {
					ui.Box(c).Height(4).WidthPercent(float32(a.updateProgress * 100)).Radius(2).Background(col.accent)
				}
			})
			ui.Row(c).Gap(8).Justify(ui.End).AlignItems(ui.Center).Children(func() {
				ui.Textf(c, "下载中 %.0f%%", a.updateProgress*100).FontSize(12).TextColor(col.muted)
			})
		default:
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				secondaryButton(c, col, nil, "稍后", close)
				if a.updateReady != nil {
					primaryButton(c, col, iconDownload, "立即更新", func() { a.installUpdate() })
				} else {
					primaryButton(c, col, iconExternal, "打开发布页", func() {
						close()
						a.openURL(a.updateInfo.url)
					})
				}
			})
		}
	})
}

func fetchLatestRelease(ctx context.Context) (releaseInfo, error) {
	if release, err := fetchLatestReleasePage(ctx); err == nil {
		return release, nil
	}
	return fetchLatestReleaseAPI(ctx)
}

// fetchLatestReleasePage reads the tag GitHub redirects the latest release
// page to, which is not subject to the API's hourly rate limit.
func fetchLatestReleasePage(ctx context.Context) (releaseInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleasePage, nil)
	if err != nil {
		return releaseInfo{}, err
	}
	req.Header.Set("User-Agent", "Qmdr/"+appVersion)
	client := &http.Client{
		Timeout: 12 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return releaseInfo{}, err
	}
	defer resp.Body.Close()
	location := resp.Header.Get("Location")
	if location == "" {
		return releaseInfo{}, fmt.Errorf("GitHub 未返回版本信息")
	}
	target, err := resp.Request.URL.Parse(location)
	if err != nil {
		return releaseInfo{}, err
	}
	const marker = "/releases/tag/"
	index := strings.Index(target.Path, marker)
	if index < 0 {
		return releaseInfo{}, fmt.Errorf("无法解析版本号")
	}
	tag, err := url.PathUnescape(target.Path[index+len(marker):])
	if err != nil || tag == "" {
		return releaseInfo{}, fmt.Errorf("无法解析版本号")
	}
	return releaseInfo{TagName: tag, HTMLURL: target.String()}, nil
}

func fetchLatestReleaseAPI(ctx context.Context) (releaseInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseAPI, nil)
	if err != nil {
		return releaseInfo{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Qmdr/"+appVersion)
	resp, err := (&http.Client{Timeout: 12 * time.Second}).Do(req)
	if err != nil {
		return releaseInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&apiErr)
		if apiErr.Message != "" {
			return releaseInfo{}, errors.New(apiErr.Message)
		}
		return releaseInfo{}, fmt.Errorf("GitHub 返回 %s", resp.Status)
	}
	var release releaseInfo
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&release); err != nil {
		return releaseInfo{}, fmt.Errorf("无法读取更新信息：%w", err)
	}
	return release, nil
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
)

const (
	appVersion        = "0.2.0"
	repoURL           = "https://github.com/rzh0504/Qmdr"
	latestReleasePage = repoURL + "/releases/latest"
	latestReleaseAPI  = "https://api.github.com/repos/rzh0504/Qmdr/releases/latest"
)

type releaseInfo struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
}

func (a *app) checkUpdates() {
	if a.updating {
		a.setToast("正在检查更新…")
		return
	}
	a.updating = true
	a.setToast("正在检查更新…")
	win := a.win

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		release, err := fetchLatestRelease(ctx)
		a.update(func() { a.updating = false })
		if err != nil {
			a.update(func() { a.toast = "检查更新失败：" + err.Error() })
			return
		}
		latest := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
		if compareVersions(latest, appVersion) <= 0 {
			a.update(func() { a.toast = "已是最新版本 v" + appVersion })
			return
		}

		detail := "当前版本 v" + appVersion + "。"
		if release.Name != "" && !strings.EqualFold(release.Name, release.TagName) {
			detail = release.Name + "\n\n" + detail
		}
		result, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:        win,
			Type:          mygo.MessageInfo,
			Title:         "发现新版本",
			Message:       "Qmdr " + latest + " 已发布",
			Detail:        detail,
			Buttons:       []string{"打开发布页", "稍后"},
			DefaultButton: 0,
			CancelButton:  1,
		})
		if err != nil {
			return
		}
		if result.Button == 0 {
			url := release.HTMLURL
			if url == "" {
				url = repoURL + "/releases/latest"
			}
			a.openURL(url)
		}
	}()
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

func compareVersions(a, b string) int {
	av, bv := versionParts(a), versionParts(b)
	n := max(len(av), len(bv))
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(av) {
			x = av[i]
		}
		if i < len(bv) {
			y = bv[i]
		}
		if x != y {
			if x > y {
				return 1
			}
			return -1
		}
	}
	return 0
}

func versionParts(v string) []int {
	v = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V"))
	if i := strings.IndexAny(v, "-+ "); i >= 0 {
		v = v[:i]
	}
	if v == "" {
		return nil
	}
	fields := strings.Split(v, ".")
	parts := make([]int, len(fields))
	for i, field := range fields {
		n, err := strconv.Atoi(field)
		if err != nil {
			n = 0
		}
		parts[i] = n
	}
	return parts
}

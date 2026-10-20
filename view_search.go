package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/qqmusic"
)

func (a *app) searchView(c *ui.Context, col colors) {
	pageHeader(c, col, "搜索下载", "搜索 QQ 音乐，选择音质后加入下载队列", func() {
		secondaryButton(c, col, iconFolder, "打开下载目录", func() {
			a.openDownloadDir()
		})
	})

	a.searchToolbar(c, col)
	a.searchResults(c, col)
}

func (a *app) searchToolbar(c *ui.Context, col colors) {
	card(c, col, func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			searchBox(c, col, &a.query, "搜索歌曲、歌手或专辑",
				func() { a.doSearch() },
				func() { a.query = ""; a.refresh() },
			).Grow(1)
			ui.Select(c, &a.quality, qualityOptions).Width(150)
			primaryButton(c, col, iconSearch, "搜索", func() {
				a.doSearch()
			})
		})

		ui.Row(c).Gap(6).AlignItems(ui.Center).Wrap().Children(func() {
			ui.Text(c, "热门搜索").FontSize(11).TextColor(col.muted)
			for _, tag := range []string{"周杰伦", "华语流行", "深夜电台", "G.E.M. 邓紫棋", "陈奕迅"} {
				tag := tag
				textLink(c, col, tag, func() {
					a.query = tag
					a.doSearch()
				})
			}
		})
	})
}

func (a *app) searchResults(c *ui.Context, col colors) {
	if !a.hasSearched {
		sectionHeader(c, col, "搜索歌曲", "输入关键词，或从热门搜索开始", nil)
		card(c, col, func() {
			emptyState(c, col, iconSearch, "还没有搜索", "在上方输入歌曲、歌手或专辑名开始搜索。")
		})
		return
	}

	subtitle := "正在搜索…"
	if !a.searching {
		if a.searchErr != "" {
			subtitle = "搜索失败"
		} else {
			subtitle = fmt.Sprintf("找到 %d 首与「%s」相关的歌曲", len(a.results), a.query)
		}
	}
	sectionHeader(c, col, "搜索结果", subtitle, func() {
		if len(a.results) > 0 && !a.searching {
			secondaryButton(c, col, iconDownload, "全部下载", func() {
				a.enqueueSongs(a.results)
			})
		}
	})

	if a.searching {
		card(c, col, func() {
			ui.Row(c).Gap(9).AlignItems(ui.Center).Padding(6).Children(func() {
				ui.Spinner(c).Size(18, 18)
				ui.Column(c).Gap(1).Children(func() {
					ui.Text(c, "正在搜索 QQ 音乐").FontSize(13).Bold().TextColor(col.text)
					ui.Text(c, "正在匹配歌曲、歌手和专辑…").FontSize(11).TextColor(col.muted)
				})
			})
		})
		return
	}

	if a.searchErr != "" {
		card(c, col, func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconAlert).FontSize(16).TextColor(col.red)
				ui.Column(c).Gap(1).Grow(1).Children(func() {
					ui.Text(c, "搜索失败").FontSize(13).Bold().TextColor(col.text)
					ui.Text(c, a.searchErr).FontSize(11).TextColor(col.muted).MaxLines(2)
				})
				secondaryButton(c, col, iconRefresh, "重试", func() {
					a.doSearch()
				})
			})
		})
		return
	}

	if len(a.results) == 0 {
		card(c, col, func() {
			emptyState(c, col, iconSearch, "没有找到相关歌曲", "换一个关键词试试，或点击上方的热门搜索。")
		})
		return
	}

	tableCard(c, col, func() {
		ui.Row(c).FillWidth().Padding(4, 10).Children(func() {
			ui.Text(c, "#").Width(24).FontSize(10).TextColor(col.muted)
			ui.Text(c, "歌曲").Grow(1).FontSize(10).TextColor(col.muted)
			ui.Text(c, "时长").Width(42).FontSize(10).TextColor(col.muted)
			ui.Box(c).Width(28)
		})
		divider(c, col)
		for i, song := range a.results {
			song := song
			songRow(c, col, song, i, true, func() {
				a.enqueueSongs([]Song{song})
			})
			if i < len(a.results)-1 {
				divider(c, col)
			}
		}
	})
}

func (a *app) doSearch() {
	query := strings.TrimSpace(a.query)
	if query == "" {
		a.setToast("请输入歌曲关键词")
		return
	}
	a.searching = true
	a.hasSearched = true
	a.searchErr = ""
	a.searchID++
	requestID := a.searchID
	a.results = nil
	a.refresh()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		songs, err := a.api.Search(ctx, query, 20)
		mapped := make([]Song, 0, len(songs))
		for _, song := range songs {
			mapped = append(mapped, songFromAPI(song))
		}
		a.update(func() {
			if a.searchID != requestID {
				return
			}
			a.searching = false
			if err != nil {
				a.searchErr = friendlyError(err)
				a.results = nil
				return
			}
			a.results = mapped
			if len(mapped) == 0 {
				a.toast = "没有找到相关歌曲"
			} else {
				a.toast = fmt.Sprintf("找到 %d 首歌曲", len(mapped))
			}
		})
	}()
}

func songFromAPI(song qqmusic.Song) Song {
	artist := strings.Join(song.Artists, " / ")
	if artist == "" {
		artist = "未知歌手"
	}
	album := song.Album
	if album == "" {
		album = "未知专辑"
	}
	title := song.Title
	if title == "" {
		title = "未知歌曲"
	}
	duration := fmt.Sprintf("%02d:%02d", song.DurationSeconds/60, song.DurationSeconds%60)
	return Song{
		Mid:      song.Mid,
		Title:    title,
		Artist:   artist,
		Album:    album,
		Duration: duration,
		Quality:  song.Quality,
		VIP:      song.VIP,
	}
}

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
	a.searchToolbar(c, col)
	a.searchResults(c, col)
}

func (a *app) searchToolbar(c *ui.Context, col colors) {
	ui.Column(c).Gap(10).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			searchBox(c, col, &a.search.query, "搜索歌曲、歌手或专辑",
				func() { a.doSearch() },
				func() { a.search.query = ""; a.refresh() },
			).Grow(1)
			selectControl(c, col, &a.quality, qualityOptions, 140, col.chip, nil)
			primaryButton(c, col, iconSearch, "搜索", func() {
				a.doSearch()
			})
		})

		ui.Row(c).Gap(2).AlignItems(ui.Center).Wrap().Children(func() {
			ui.Text(c, "试试").FontSize(12).TextColor(col.faint).Padding(0, 6, 0, 0)
			for _, tag := range []string{"周杰伦", "华语流行", "深夜电台", "G.E.M. 邓紫棋", "陈奕迅"} {
				tag := tag
				textLink(c, col, tag, func() {
					a.search.query = tag
					a.doSearch()
				})
			}
		})
	})
}

const searchPageSize = 20

func (a *app) searchResults(c *ui.Context, col colors) {
	if !a.search.searched {
		ui.Column(c).Grow(1).FillWidth().Center().Children(func() {
			emptyState(c, col, iconSearch, "搜索", "支持歌曲、歌手、专辑搜索")
		})
		return
	}

	ui.Column(c).Gap(8).Children(func() {
		sectionHeader(c, col, "搜索结果", func() {
			if !a.search.loading && a.search.err == "" {
				if a.search.hasMore {
					ui.Textf(c, "已加载 %d 首", len(a.search.results)).FontSize(12).TextColor(col.faint)
				} else {
					ui.Textf(c, "%d 首", len(a.search.results)).FontSize(12).TextColor(col.faint)
				}
			}
			if len(a.search.results) > 0 && !a.search.loading {
				secondaryButton(c, col, iconDownload, "全部下载", func() {
					a.enqueueSongs(a.search.results)
				})
			}
		})

		switch {
		case a.search.loading:
			softCard(c, col, func() {
				ui.Row(c).Gap(9).AlignItems(ui.Center).Children(func() {
					ui.Spinner(c).Size(15, 15)
					ui.Text(c, "正在搜索…").FontSize(12).TextColor(col.muted)
				})
			})
		case a.search.err != "":
			softCard(c, col, func() {
				ui.Row(c).Gap(9).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconAlert).Size(15, 15).TextColor(col.red)
					ui.Text(c, a.search.err).FontSize(12).TextColor(col.muted).Grow(1).MaxLines(2)
					secondaryButton(c, col, iconRefresh, "重试", func() {
						a.doSearch()
					})
				})
			})
		case len(a.search.results) == 0:
			emptyState(c, col, iconSearch, "没有找到相关歌曲", "换一个关键词试试。")
		default:
			sheet(c, col, func() {
				for i, song := range a.search.results {
					song := song
					row := a.songRow(c, col, song, songRowOptions{
						index: i,
						onDownload: func() {
							a.enqueueSongs([]Song{song})
						},
					})
					a.openSongMenu(row, song, false)
				}
			})
			if a.search.hasMore || a.search.loadingMore {
				ui.Row(c).FillWidth().Height(36).Justify(ui.Center).
					AlignItems(ui.Center).Gap(6).Children(func() {
					if a.search.loadingMore {
						ui.Spinner(c).Size(13, 13)
						ui.Text(c, "正在加载更多…").FontSize(12).TextColor(col.muted)
					} else {
						secondaryButton(c, col, iconChevronDown, "加载更多", a.loadMoreSearch)
					}
				})
			}
		}
	})
}

func (a *app) doSearch() {
	query := strings.TrimSpace(a.search.query)
	if query == "" {
		a.setToast("请输入歌曲关键词")
		return
	}
	var page qqmusic.SearchPage
	a.search.loading = true
	a.search.searched = true
	a.search.err = ""
	a.search.results = nil
	a.search.searchID = ""
	a.search.page = 0
	a.search.hasMore = false
	a.search.loadingMore = false
	a.refresh()
	a.async(&a.search.slot, 20*time.Second,
		func(ctx context.Context) error {
			fetched, err := a.api.Search(ctx, query, "", 1, searchPageSize)
			page = fetched
			return err
		},
		func(err error) {
			a.search.loading = false
			if err != nil {
				a.search.err = friendlyError(err)
				return
			}
			a.search.results = mapSearchSongs(page.Songs)
			a.search.searchID = page.SearchID
			a.search.page = 1
			a.search.hasMore = page.HasMore
			if len(page.Songs) == 0 {
				a.toast = "没有找到相关歌曲"
			}
		})
}

func (a *app) loadMoreSearch() {
	if a.search.loading || a.search.loadingMore || !a.search.hasMore {
		return
	}
	query := strings.TrimSpace(a.search.query)
	searchID := a.search.searchID
	if query == "" || searchID == "" {
		return
	}
	next := a.search.page + 1
	var page qqmusic.SearchPage
	a.search.loadingMore = true
	a.refresh()
	a.async(&a.search.slot, 20*time.Second,
		func(ctx context.Context) error {
			fetched, err := a.api.Search(ctx, query, searchID, next, searchPageSize)
			page = fetched
			return err
		},
		func(err error) {
			a.search.loadingMore = false
			if err != nil {
				a.toast = "加载更多失败：" + friendlyError(err)
				return
			}
			a.search.page = next
			a.search.hasMore = page.HasMore
			a.search.results = appendSongs(a.search.results, mapSearchSongs(page.Songs))
		})
}

func mapSearchSongs(songs []qqmusic.Song) []Song {
	mapped := make([]Song, 0, len(songs))
	for _, song := range songs {
		mapped = append(mapped, songFromAPI(song))
	}
	return mapped
}

// appendSongs adds songs not already in base, keeping first-seen order, as
// pages of a search can overlap.
func appendSongs(base, more []Song) []Song {
	seen := make(map[string]bool, len(base)+len(more))
	for _, song := range base {
		if song.Mid != "" {
			seen[song.Mid] = true
		}
	}
	for _, song := range more {
		if song.Mid != "" {
			if seen[song.Mid] {
				continue
			}
			seen[song.Mid] = true
		}
		base = append(base, song)
	}
	return base
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
		AlbumMid: song.AlbumMid,
		Duration: duration,
		Quality:  song.Quality,
		VIP:      song.VIP,
	}
}

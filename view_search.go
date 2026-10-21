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
				ui.Textf(c, "%d 首", len(a.search.results)).FontSize(12).TextColor(col.faint)
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
					a.openSongMenu(row, song)
				}
			})
		}
	})
}

func (a *app) doSearch() {
	query := strings.TrimSpace(a.search.query)
	if query == "" {
		a.setToast("请输入歌曲关键词")
		return
	}
	var found []Song
	a.search.loading = true
	a.search.searched = true
	a.search.err = ""
	a.search.results = nil
	a.refresh()
	a.async(&a.search.slot, 20*time.Second,
		func(ctx context.Context) error {
			songs, err := a.api.Search(ctx, query, 20)
			if err != nil {
				return err
			}
			found = make([]Song, 0, len(songs))
			for _, song := range songs {
				found = append(found, songFromAPI(song))
			}
			return nil
		},
		func(err error) {
			a.search.loading = false
			if err != nil {
				a.search.err = friendlyError(err)
				return
			}
			a.search.results = found
			if len(found) == 0 {
				a.toast = "没有找到相关歌曲"
			} else {
				a.toast = fmt.Sprintf("找到 %d 首歌曲", len(found))
			}
		})
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

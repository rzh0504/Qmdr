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
	a.loadExplore(false)
	a.searchToolbar(c, col)
	if a.search.searched {
		a.searchResults(c, col)
		return
	}
	a.exploreView(c, col)
}

func (a *app) searchToolbar(c *ui.Context, col colors) {
	ui.Column(c).Gap(10).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			searchBox(c, col, &a.search.query, "搜索歌曲、歌手或专辑",
				func() { a.doSearch() },
				func() { a.clearSearch() },
			).Grow(1)
			selectControl(c, col, &a.quality, qualityOptions, 140, col.chip, nil)
			primaryButton(c, col, iconSearch, "搜索", func() {
				a.doSearch()
			})
		})

		tags := a.search.hotSearches
		if len(tags) == 0 {
			tags = []string{"周杰伦", "华语流行", "深夜电台", "G.E.M. 邓紫棋", "陈奕迅"}
		}
		ui.Row(c).Gap(2).AlignItems(ui.Center).Wrap().Children(func() {
			ui.Text(c, "试试").FontSize(12).TextColor(col.faint).Padding(0, 6, 0, 0)
			for _, tag := range tags {
				tag := tag
				textLink(c, col, tag, func() {
					a.search.query = tag
					a.doSearch()
				})
			}
		})
	})
}

const (
	searchPageSize          = 20
	searchLoadMoreThreshold = 120
	hotSearchCount          = 8
	exploreChartLimit       = 20
)

// exploreChartSpecs are the toplists on the search page, in order.
var exploreChartSpecs = []struct {
	id   int
	name string
}{
	{26, "热歌榜"},
	{62, "飙升榜"},
	{427, "新歌榜"},
}

// loadExplore fills the page shown before a search with hot terms and the
// three toplist cards. It fetches once per session, and again when asked.
func (a *app) loadExplore(force bool) {
	if a.search.exploreLoading || (a.search.exploreLoaded && !force) {
		return
	}
	previous := append([]chartPreview(nil), a.search.charts...)
	a.search.exploreLoading = true
	var hot []string
	var charts []chartPreview
	a.async(&a.search.exploreSlot, 30*time.Second,
		func(ctx context.Context) error {
			hot, _ = a.api.HotSearches(ctx, hotSearchCount)
			charts = make([]chartPreview, 0, len(exploreChartSpecs))
			for _, spec := range exploreChartSpecs {
				chart, err := a.api.Chart(ctx, spec.id, exploreChartLimit)
				if err != nil {
					charts = append(charts, chartPreview{id: spec.id, name: spec.name, err: friendlyError(err)})
					continue
				}
				name := chart.Title
				if name == "" {
					name = spec.name
				}
				charts = append(charts, chartPreview{
					id:    spec.id,
					name:  name,
					songs: mapSearchSongs(chart.Songs),
				})
			}
			return nil
		},
		func(err error) {
			a.search.exploreLoading = false
			a.search.exploreLoaded = true
			if err != nil {
				return
			}
			if len(hot) > 0 {
				a.search.hotSearches = hot
			}
			old := make(map[int]chartPreview, len(previous))
			for _, chart := range previous {
				old[chart.id] = chart
			}
			failed := 0
			for i := range charts {
				if charts[i].err == "" {
					continue
				}
				if prev, ok := old[charts[i].id]; ok && len(prev.songs) > 0 {
					charts[i] = prev
					continue
				}
				failed++
			}
			a.search.charts = charts
			if failed > 0 && len(previous) > 0 {
				if failed == len(charts) {
					a.toast = "榜单刷新失败"
				} else {
					a.toast = "部分榜单刷新失败"
				}
			}
		})
}

func (a *app) exploreView(c *ui.Context, col colors) {
	if len(a.search.charts) == 0 {
		if a.search.exploreLoading {
			softCard(c, col, func() {
				ui.Row(c).Gap(9).AlignItems(ui.Center).Children(func() {
					ui.Spinner(c).Size(15, 15)
					ui.Text(c, "正在加载热门榜单…").FontSize(12).TextColor(col.muted)
				})
			})
			return
		}
		softCard(c, col, func() {
			ui.Row(c).Gap(9).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconAlert).Size(15, 15).TextColor(col.red)
				ui.Text(c, "热门内容暂时不可用").FontSize(12).TextColor(col.muted).Grow(1).MaxLines(2)
				secondaryButton(c, col, iconRefresh, "重试", func() {
					a.loadExplore(true)
				})
			})
		})
		return
	}

	sectionHeader(c, col, "热门榜单", nil)
	ui.Row(c).Gap(14).AlignItems(ui.Start).Children(func() {
		for _, chart := range a.search.charts {
			chart := chart
			a.chartCard(c, col, chart)
		}
	})
}

func (a *app) chartCard(c *ui.Context, col colors, chart chartPreview) {
	ui.Column(c).Grow(1).MinWidth(0).Gap(6).Children(func() {
		ui.Text(c, chart.name).FontSize(13).FontWeight(600).TextColor(col.text).SingleLine()
		if chart.err != "" {
			ui.Text(c, chart.err).FontSize(11).TextColor(col.muted).MaxLines(2)
			textLink(c, col, "重试", func() {
				a.loadExplore(true)
			})
			return
		}
		if len(chart.songs) == 0 {
			ui.Text(c, "暂无内容").FontSize(11).TextColor(col.faint)
			return
		}
		sheet(c, col, func() {
			for i, song := range chart.songs {
				song := song
				a.chartSongRow(c, col, song, i)
			}
		})
	})
}

func (a *app) chartSongRow(c *ui.Context, col colors, song Song, index int) {
	row := ui.Row(c).Gap(7).Padding(4, 6).Height(42).Radius(7).AlignItems(ui.Center)
	if row.Hovered() {
		row.Background(col.rowSelected)
		row = chosen(row, col)
	}
	row.Children(func() {
		ui.Textf(c, "%02d", index+1).Width(17).FontSize(10).TextColor(col.faint)
		a.cover(c, col, song.AlbumMid, 26)
		ui.Column(c).Gap(1).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, song.Title).FontSize(12).FontWeight(400).TextColor(col.text).SingleLine().Ellipsis("…")
			ui.Text(c, song.Artist).FontSize(10).TextColor(col.muted).SingleLine().Ellipsis("…")
		})
	})
	a.openSongMenu(row, song, false)
}

func (a *app) searchResults(c *ui.Context, col colors) {
	if a.search.hasMore && !a.search.loading && !a.search.loadingMore &&
		a.search.scroll.MaxY > 0 &&
		a.search.scroll.Y >= a.search.scroll.MaxY-searchLoadMoreThreshold {
		a.loadMoreSearch()
	}

	ui.Column(c).Gap(8).Children(func() {
		sectionHeader(c, col, "搜索结果", func() {
			if !a.search.loading && a.search.err == "" && !a.search.hasMore {
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
					a.openSongMenu(row, song, false)
				}
			})
			if a.search.loadingMore {
				ui.Row(c).FillWidth().Height(36).Justify(ui.Center).
					AlignItems(ui.Center).Gap(6).Children(func() {
					ui.Spinner(c).Size(13, 13)
					ui.Text(c, "正在加载更多…").FontSize(12).TextColor(col.muted)
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
	a.search.scroll.Y = 0
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

// clearSearch empties the search box and removes its results, cancelling
// a search still in flight.
func (a *app) clearSearch() {
	a.search.slot.reset()
	a.search.query = ""
	a.search.searched = false
	a.search.loading = false
	a.search.err = ""
	a.search.results = nil
	a.search.searchID = ""
	a.search.page = 0
	a.search.hasMore = false
	a.search.loadingMore = false
	a.search.scroll.Y = 0
	a.refresh()
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

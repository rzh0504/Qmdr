package main

import (
	"time"

	"github.com/rzh0504/qmdr/internal/explorecache"
)

// restoreExploreCache fills the search page from disk before its first
// render, so a same-day cache appears without a loading frame.
func (a *app) restoreExploreCache() {
	cache, ok := explorecache.Load()
	if !ok {
		return
	}
	byID := make(map[int]explorecache.Chart, len(cache.Charts))
	for _, chart := range cache.Charts {
		if len(chart.Songs) > 0 {
			byID[chart.ID] = chart
		}
	}
	charts := make([]chartPreview, 0, len(exploreChartSpecs))
	for _, spec := range exploreChartSpecs {
		chart, ok := byID[spec.id]
		if !ok {
			continue
		}
		name := chart.Name
		if name == "" {
			name = spec.name
		}
		charts = append(charts, chartPreview{
			id:    spec.id,
			name:  name,
			songs: chart.Songs,
		})
	}
	if len(charts) == 0 {
		return
	}
	a.search.hotSearches = cache.HotSearches
	a.search.charts = charts
	a.search.exploreDay = cache.Day
}

func (a *app) saveExploreCache(day string) {
	path := explorecache.Path()
	if path == "" {
		return
	}
	charts := make([]explorecache.Chart, 0, len(exploreChartSpecs))
	for _, chart := range a.search.charts {
		if len(chart.songs) == 0 {
			return
		}
		charts = append(charts, explorecache.Chart{
			ID:    chart.id,
			Name:  chart.name,
			Songs: chart.songs,
		})
	}
	if len(charts) != len(exploreChartSpecs) {
		return
	}
	cache := explorecache.Cache{
		Day:         day,
		HotSearches: append([]string(nil), a.search.hotSearches...),
		Charts:      charts,
		UpdatedAt:   time.Now(),
	}
	a.dispatchDisk(func() {
		if err := explorecache.Save(path, cache); err != nil {
			a.update(func() {
				a.appendLog("榜单缓存保存失败："+friendlyError(err), "warning")
			})
		}
	})
}

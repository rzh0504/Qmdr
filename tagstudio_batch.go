package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/fsutil"
	"github.com/rzh0504/qmdr/internal/qqmusic"
	"github.com/rzh0504/qmdr/internal/tag"
)

// Batch fill statuses shown in the table's status column.
const (
	batchUpdated   = "已补全"
	batchNoop      = "无更新"
	batchUnmatched = "未匹配"
	batchFailed    = "失败"
)

// maxBatchSearchFails trips the auto-match circuit breaker: after this many
// consecutive search failures the batch stops auto matching and marks the
// remaining entries, instead of hammering a rate-limited endpoint.
const maxBatchSearchFails = 3

// batchPause sleeps a randomized short interval between batch API calls, so
// a large batch never hammers QQ Music fast enough to trip risk control.
func batchPause() {
	time.Sleep(time.Duration(300+rand.Intn(400)) * time.Millisecond)
}

// tagBatchItem snapshots one picked entry for the batch fill, so the worker
// goroutine never touches the live entry list. query is the auto search
// built from the entry's own info, used when it has no song MID yet.
type tagBatchItem struct {
	path     string
	songMID  string
	albumMID string
	query    string
	title    string
	artist   string
	album    string
}

// enterTagBatch switches the page into the batch table; a dirty draft must
// be saved or reverted first, or the batch could clobber it.
func (a *app) enterTagBatch() {
	s := &a.tagStudio
	if s.dirty {
		a.setToast("当前有未保存的修改，请先保存或还原")
		return
	}
	s.batchMode = true
	s.picked = nil
	s.batchResults = nil
	a.refresh()
}

func (a *app) exitTagBatch() {
	s := &a.tagStudio
	s.batchMode = false
	s.picked = nil
	s.batchResults = nil
	a.refresh()
}

func (a *app) toggleTagPicked(entry *tagEntry) {
	s := &a.tagStudio
	key := pathKey(entry.path)
	if s.picked[key] {
		delete(s.picked, key)
	} else {
		if s.picked == nil {
			s.picked = map[string]bool{}
		}
		s.picked[key] = true
	}
	a.refresh()
}

func (a *app) tagPickedCount() int {
	s := &a.tagStudio
	count := 0
	for _, entry := range s.activeEntries() {
		if s.picked[pathKey(entry.path)] {
			count++
		}
	}
	return count
}

func (a *app) tagAllPicked() bool {
	s := &a.tagStudio
	entries := s.activeEntries()
	if len(entries) == 0 {
		return false
	}
	for _, entry := range entries {
		if !s.picked[pathKey(entry.path)] {
			return false
		}
	}
	return true
}

func (a *app) toggleAllTagPicked() {
	s := &a.tagStudio
	if a.tagAllPicked() {
		s.picked = nil
		a.refresh()
		return
	}
	s.picked = map[string]bool{}
	for _, entry := range s.activeEntries() {
		s.picked[pathKey(entry.path)] = true
	}
	a.refresh()
}

func (a *app) openTagBatchConfirm() {
	s := &a.tagStudio
	if s.batchBusy {
		a.setToast("批量补全进行中，请稍候")
		return
	}
	if len(a.tagBatchItems()) == 0 {
		a.setToast("勾选的歌曲都没有关联 QQ 音乐 ID")
		return
	}
	s.batchConfirm = true
	a.refresh()
}

func (a *app) tagBatchConfirmModal(c *ui.Context, col colors) {
	a.modal(c, col, &a.tagStudio.batchConfirm, "批量补全", 420, func(close func()) {
		ui.Row(c).Gap(10).AlignItems(ui.Start).Children(func() {
			ui.Icon(c, iconSparkles).Size(20, 20).TextColor(col.accent)
			ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
				ui.Textf(c, "将为 %d 个文件从 QQ 音乐补全标签信息。", len(a.tagBatchItems())).
					FontSize(14).TextColor(col.text).LineHeight(1.35)
				ui.Text(c, "只填写空缺字段（标题、艺人、专辑、日期、流派、语言、作曲、作词）并直接写入文件；未关联的歌曲会先按文件信息自动匹配，匹配不确定的跳过。").
					FontSize(12).TextColor(col.muted).LineHeight(1.35)
			})
		})
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			secondaryButton(c, col, nil, "取消", close)
			primaryButton(c, col, nil, "开始补全", func() {
				close()
				a.startTagBatchFill()
			})
		})
	})
}

// tagBatchItems snapshots the picked entries into work items, in list order.
// Entries without a song MID carry the auto search query built from their
// own info — the batch matches them before filling.
func (a *app) tagBatchItems() []tagBatchItem {
	s := &a.tagStudio
	items := []tagBatchItem{}
	for _, entry := range s.activeEntries() {
		if !s.picked[pathKey(entry.path)] {
			continue
		}
		items = append(items, tagBatchItem{
			path:     entry.path,
			songMID:  entry.songMID,
			albumMID: entry.albumMID,
			query:    a.tagMatchQuery(&entry),
			title:    entry.displayTitle(),
			artist:   entry.displayArtist(),
			album:    entry.displayAlbum(),
		})
	}
	return items
}

func (a *app) startTagBatchFill() {
	s := &a.tagStudio
	if s.batchBusy {
		return
	}
	items := a.tagBatchItems()
	if len(items) == 0 {
		a.setToast("没有勾选的歌曲")
		return
	}
	s.batchBusy = true
	s.batchDone = 0
	s.batchTotal = len(items)
	s.batchResults = map[string]string{}
	tally := struct{ updated, noop, unmatched, failed int }{}
	a.refresh()
	go func() {
		// 单轮批量的搜索缓存与失败熔断状态，仅本 goroutine 使用。
		searchCache := map[string][]qqmusic.Song{}
		searchFailStreak := 0
		for _, item := range items {
			songMID, albumMID := item.songMID, item.albumMID
			autoMatched := false
			if songMID == "" && searchFailStreak < maxBatchSearchFails {
				if song, ok := a.batchAutoMatch(item, searchCache, &searchFailStreak); ok {
					songMID, albumMID = song.Mid, song.AlbumMid
					autoMatched = true
				}
			}
			status, info := batchNoop, tag.Info{}
			if songMID != "" {
				filled := item
				filled.songMID, filled.albumMID = songMID, albumMID
				status, info = a.batchFillOne(filled)
			} else {
				status = batchUnmatched
			}
			a.update(func() {
				s.batchDone++
				switch status {
				case batchUpdated:
					tally.updated++
					s.batchResults[pathKey(item.path)] = batchUpdated
				case batchNoop:
					tally.noop++
					s.batchResults[pathKey(item.path)] = batchNoop
				case batchUnmatched:
					tally.unmatched++
					s.batchResults[pathKey(item.path)] = batchUnmatched
				default:
					tally.failed++
					s.batchResults[pathKey(item.path)] = batchFailed
				}
				if songMID != "" && (autoMatched || status == batchUpdated) {
					a.bindTagEntryByPath(item.path, songMID, albumMID)
				}
				if status == batchUpdated {
					a.refreshTagEntryAfterBatch(item.path, info)
				}
				a.refresh()
			})
			// 每个文件处理完都稍作停顿，降低连续请求的密度。
			batchPause()
		}
		a.update(func() {
			s.batchBusy = false
			note := ""
			if searchFailStreak >= maxBatchSearchFails {
				note = "（搜索连续失败，已暂停自动匹配）"
			}
			a.toast = fmt.Sprintf("批量补全完成：已补全 %d · 无更新 %d · 未匹配 %d · 失败 %d%s",
				tally.updated, tally.noop, tally.unmatched, tally.failed, note)
			a.refresh()
		})
	}()
}

// batchAutoMatch searches by the item's own info and returns the first
// confidently matching result. Identical queries reuse one search, and a
// streak of failures pauses further auto matching for the rest of the run.
func (a *app) batchAutoMatch(item tagBatchItem, cache map[string][]qqmusic.Song, failStreak *int) (qqmusic.Song, bool) {
	if songs, ok := cache[item.query]; ok {
		return tagBatchPick(item, songs)
	}
	if *failStreak >= maxBatchSearchFails {
		return qqmusic.Song{}, false
	}
	// 搜索前先节流，避免与上一文件的请求贴得太近。
	batchPause()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	page, err := a.api.Search(ctx, item.query, "", 1, 5)
	if err != nil {
		log.Printf("batch auto match %s: %v", item.path, err)
		*failStreak++
		return qqmusic.Song{}, false
	}
	*failStreak = 0
	cache[item.query] = page.Songs
	return tagBatchPick(item, page.Songs)
}

// tagBatchPick returns the first search result confidently matching the
// item's own title and artist.
func tagBatchPick(item tagBatchItem, songs []qqmusic.Song) (qqmusic.Song, bool) {
	for _, song := range songs {
		if tagBatchMatchOK(item, song) {
			return song, true
		}
	}
	return qqmusic.Song{}, false
}

// tagNormKey folds a string for loose comparison: bracketed noise stripped,
// lowercased, whitespace removed.
func tagNormKey(s string) string {
	s = tagMatchNoise.ReplaceAllString(strings.TrimSpace(s), "")
	return strings.ToLower(strings.Join(strings.Fields(s), ""))
}

// tagBatchMatchOK decides whether a search result may be bound to the entry
// automatically: titles must be equal, or overlap both ways with matching
// artists — anything looser needs the user's judgement.
func tagBatchMatchOK(item tagBatchItem, song qqmusic.Song) bool {
	titleItem, titleSong := tagNormKey(item.title), tagNormKey(song.Title)
	if titleItem == "" || titleSong == "" {
		return false
	}
	artistItem := tagNormKey(item.artist)
	var artistSong strings.Builder
	for _, name := range song.Artists {
		artistSong.WriteString(tagNormKey(name))
	}
	artistMatch := artistSong.String() != "" &&
		(strings.Contains(artistSong.String(), artistItem) ||
			strings.Contains(artistItem, artistSong.String()))
	if titleSong == titleItem {
		// 标题一致即可绑定；文件没有艺人信息时也不例外。
		return artistItem == "" || artistMatch
	}
	// 标题不完全一致：必须同时有艺人信息且艺人一致，模糊绑定风险太高。
	if artistItem == "" || !artistMatch {
		return false
	}
	return strings.Contains(titleSong, titleItem) || strings.Contains(titleItem, titleSong)
}

// bindTagEntryByPath records a matched song's MIDs on every entry with that
// path, so the association survives a restart (both lists are scanned).
func (a *app) bindTagEntryByPath(path, songMID, albumMID string) {
	s := &a.tagStudio
	for _, entries := range [][]tagEntry{s.historyEntries, s.fileEntries} {
		for i := range entries {
			if samePath(entries[i].path, path) {
				entries[i].songMID = songMID
				entries[i].albumMID = albumMID
			}
		}
	}
}

// batchFillOne fills one file's empty tag fields from QQ Music and writes it
// back, returning the batch status and the written tags for the entry cache.
func (a *app) batchFillOne(item tagBatchItem) (string, tag.Info) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	info, err := tag.Read(item.path)
	if err != nil {
		log.Printf("batch fill %s: %v", item.path, err)
		return batchFailed, tag.Info{}
	}
	meta, err := a.api.SongMeta(ctx, item.songMID)
	if err != nil {
		log.Printf("batch fill meta %s: %v", item.path, err)
		meta = qqmusic.SongMeta{}
	}
	credits, err := a.api.SongCredits(ctx, item.songMID)
	if err != nil {
		log.Printf("batch fill credits %s: %v", item.path, err)
		credits = qqmusic.SongCredits{}
	}
	if applyTagBatchFill(&info, meta, credits, item.title, item.artist, item.album) == 0 {
		return batchNoop, tag.Info{}
	}
	if err := tag.Write(item.path, info); err != nil {
		log.Printf("batch write %s: %v", item.path, err)
		return batchFailed, tag.Info{}
	}
	return batchUpdated, info
}

// applyTagBatchFill fills the empty fields of info from QQ Music metadata
// and the entry's known title/artist/album, and reports how many fields were
// filled. Non-empty fields are never touched.
func applyTagBatchFill(info *tag.Info, meta qqmusic.SongMeta, credits qqmusic.SongCredits, title, artist, album string) int {
	filled := 0
	fill := func(dst *string, value string) {
		value = strings.TrimSpace(value)
		if strings.TrimSpace(*dst) == "" && value != "" {
			*dst = value
			filled++
		}
	}
	fill(&info.Title, title)
	fill(&info.Artist, artist)
	fill(&info.Album, album)
	fill(&info.Date, meta.ReleaseDate)
	fill(&info.Genre, meta.Genre)
	fill(&info.Language, meta.Language)
	fill(&info.Composer, credits.Composer)
	fill(&info.Lyricist, credits.Lyricist)
	return filled
}

// refreshTagEntryAfterBatch syncs entries whose file the batch just wrote.
// Already-loaded entries refresh their cached tags (the cover bytes are
// preserved by the fill, so their bitmaps stay valid).
func (a *app) refreshTagEntryAfterBatch(path string, info tag.Info) {
	s := &a.tagStudio
	for _, entries := range [][]tagEntry{s.historyEntries, s.fileEntries} {
		for i := range entries {
			if !samePath(entries[i].path, path) || !entries[i].loaded {
				continue
			}
			entries[i].size = fsutil.FileSize(path)
			if !bytes.Equal(entries[i].info.Cover, info.Cover) {
				entries[i].cover = tagCoverBitmap(info.Cover)
			}
			entries[i].info = info
			entries[i].err = ""
			entries[i].missing = false
		}
	}
}

// tagBatchView is the batch mode workbench: a toolbar with the source and
// actions above a full-width table of the current entries.
func (a *app) tagBatchView(c *ui.Context, col colors) {
	s := &a.tagStudio
	ui.Column(c).Fill().Gap(10).Children(func() {
		// 顶对齐：来源切换控件与普通模式的行顶完全一致，切换批量模式时
		// tab 不会因工具条被搜索框撑高而往下跳。
		ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Start).Children(func() {
			a.tagStudioSourceSwitch(c, col)
			searchBox(c, col, &s.filter, "筛选标题、艺人或路径", nil, func() {
				s.filter = ""
				a.refresh()
			})
			ui.Spacer(c)
			if len(a.tagBatchItems()) > 0 {
				primaryButton(c, col, nil, "开始补全", a.openTagBatchConfirm)
			} else {
				primaryButtonDisabled(c, col, nil, "开始补全")
			}
		})
		// 表头勾选列即全选：与数据行的勾选框同列，点击切换全选。
		allLabel := a.tagAllPickedLabel()
		ui.Row(c).FillWidth().Gap(10).Padding(0, 10).AlignItems(ui.Center).Children(func() {
			headerCheck := ui.ButtonBase(c).Width(36).Height(24).Center().
				Cursor(ui.CursorPointer).Label(allLabel).Tooltip(allLabel)
			headerCheck.Children(func() {
				songSelectionMark(c, col, a.tagAllPicked())
			})
			if headerCheck.Clicked() {
				a.toggleAllTagPicked()
			}
			ui.Text(c, "标题").FontSize(12).TextColor(col.faint).Grow(1)
			ui.Text(c, "艺人").FontSize(12).TextColor(col.faint).Width(150)
			ui.Text(c, "专辑").FontSize(12).TextColor(col.faint).Width(150)
			ui.Text(c, "状态").FontSize(12).TextColor(col.faint).Width(130)
		})
		// 表格行无行距无圆角：连续选中时呈现为一整块。
		panel := ui.Column(c).Grow(1).MinHeight(0).FillWidth().Padding(4)
		smoothFlat(panel, col, 12, col.chip)
		panel.Children(func() {
			ui.Scroll(c).Key("tag-batch").Grow(1).MinHeight(0).FillWidth().Children(func() {
				matched := a.matchedTagEntries()
				for i, index := range matched {
					// 邻行的选中态决定本行圆角：连续选中块首/尾保留外圈
					// 圆角，中间行直角，整块呈现为一个整体。
					prevPicked := i > 0 && a.tagPickedAt(matched[i-1])
					nextPicked := i < len(matched)-1 && a.tagPickedAt(matched[i+1])
					a.tagBatchRow(c, col, index, prevPicked, nextPicked)
				}
			})
		})
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Textf(c, "共 %d 个文件 · 已选 %d", len(s.activeEntries()), a.tagPickedCount()).
				FontSize(11).TextColor(col.faint).Padding(0, 4)
		})
	})
}

func (a *app) tagAllPickedLabel() string {
	if a.tagAllPicked() {
		return "取消全选"
	}
	return "全选"
}

func (a *app) tagPickedAt(index int) bool {
	s := &a.tagStudio
	entry := s.activeEntry(index)
	return entry != nil && s.picked[pathKey(entry.path)]
}

// tagBatchRow is one table row: click toggles the pick, the status column
// shows what the batch would do or did with the file. prevPicked/nextPicked
// drive the row's corners, so a run of selected rows reads as one block.
func (a *app) tagBatchRow(c *ui.Context, col colors, index int, prevPicked, nextPicked bool) {
	s := &a.tagStudio
	entry := &s.activeEntries()[index]
	picked := s.picked[pathKey(entry.path)]
	tl, tr, br, bl := float32(8), float32(8), float32(8), float32(8)
	if picked {
		if prevPicked {
			tl, tr = 0, 0
		}
		if nextPicked {
			br, bl = 0, 0
		}
	}
	row := ui.ButtonBase(c).FillWidth().Height(44).Padding(0, 10).Gap(10).
		Radius(tl, tr, br, bl).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(entry.displayTitle())
	if picked {
		row.Background(col.rowSelected)
	} else if row.Hovered() {
		row.Background(col.sheetHover)
	}
	if row.Clicked() {
		a.toggleTagPicked(entry)
	}
	status, statusColor := a.tagBatchStatus(entry, col)
	row.Children(func() {
		ui.Row(c).Width(36).Center().Children(func() {
			songSelectionMark(c, col, picked)
		})
		ui.Text(c, entry.displayTitle()).FontSize(13).TextColor(col.text).
			SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
		ui.Text(c, entry.displayArtist()).FontSize(12).TextColor(col.muted).
			SingleLine().Ellipsis("…").Width(150)
		ui.Text(c, entry.displayAlbum()).FontSize(12).TextColor(col.muted).
			SingleLine().Ellipsis("…").Width(150)
		ui.Text(c, status).FontSize(11).TextColor(statusColor).
			SingleLine().Width(130)
	})
}

// tagBatchStatus is the table's status column: what the batch would do or
// did with this entry.
func (a *app) tagBatchStatus(entry *tagEntry, col colors) (string, ui.Color) {
	if entry.missing {
		return "文件缺失", col.red
	}
	if note, ok := a.tagStudio.batchResults[pathKey(entry.path)]; ok {
		switch note {
		case batchUpdated:
			return note, col.green
		case batchFailed:
			return note, col.red
		case batchUnmatched:
			return note, col.gold
		}
		return note, col.muted
	}
	if entry.songMID == "" {
		return "未关联", col.faint
	}
	return "就绪", col.muted
}

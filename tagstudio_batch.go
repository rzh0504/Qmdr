package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/fsutil"
	"github.com/rzh0504/qmdr/internal/qqmusic"
	"github.com/rzh0504/qmdr/internal/tag"
)

// Batch fill statuses shown in the table's status column.
const (
	batchUpdated = "已补全"
	batchNoop    = "无更新"
	batchFailed  = "失败"
)

// tagBatchItem snapshots one picked entry for the batch fill, so the worker
// goroutine never touches the live entry list.
type tagBatchItem struct {
	path    string
	songMID string
	title   string
	artist  string
	album   string
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
				ui.Text(c, "只填写空缺字段（标题、艺人、专辑、日期、流派、语言、作曲、作词）并直接写入文件。").
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

// tagBatchItems snapshots the picked entries into work items: entries with a
// song MID, in list order.
func (a *app) tagBatchItems() []tagBatchItem {
	s := &a.tagStudio
	items := []tagBatchItem{}
	for _, entry := range s.activeEntries() {
		if !s.picked[pathKey(entry.path)] || entry.songMID == "" {
			continue
		}
		items = append(items, tagBatchItem{
			path:    entry.path,
			songMID: entry.songMID,
			title:   entry.displayTitle(),
			artist:  entry.displayArtist(),
			album:   entry.displayAlbum(),
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
		a.setToast("勾选的歌曲都没有关联 QQ 音乐 ID")
		return
	}
	s.batchBusy = true
	s.batchDone = 0
	s.batchTotal = len(items)
	s.batchResults = map[string]string{}
	tally := struct{ updated, noop, failed int }{}
	a.refresh()
	go func() {
		for _, item := range items {
			status, info := a.batchFillOne(item)
			a.update(func() {
				s.batchDone++
				switch status {
				case batchUpdated:
					tally.updated++
					s.batchResults[pathKey(item.path)] = batchUpdated
				case batchNoop:
					tally.noop++
					s.batchResults[pathKey(item.path)] = batchNoop
				default:
					tally.failed++
					s.batchResults[pathKey(item.path)] = batchFailed
				}
				if status == batchUpdated {
					a.refreshTagEntryAfterBatch(item.path, info)
				}
				a.refresh()
			})
		}
		a.update(func() {
			s.batchBusy = false
			a.toast = fmt.Sprintf("批量补全完成：已补全 %d · 无更新 %d · 失败 %d",
				tally.updated, tally.noop, tally.failed)
			a.refresh()
		})
	}()
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

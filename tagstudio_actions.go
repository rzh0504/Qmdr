package main

// Tag Studio 草稿生命周期：加载/显示条目、切换守卫与丢弃确认、保存/还原
// 及其与历史、队列、数据库的同步，以及数字字段的解析辅助。

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/fsutil"
	"github.com/rzh0504/qmdr/internal/tag"
)

const maxTagCoverBytes = 8 << 20

// requestTagStudioAction runs action after confirming that unsaved edits may
// be dropped.
func (a *app) requestTagStudioAction(action func()) {
	s := &a.tagStudio
	if s.saving {
		a.setToast("正在保存，请稍候")
		return
	}
	if s.dirty {
		s.pendingAction = action
		s.confirmDiscard = true
		a.refresh()
		return
	}
	action()
	a.refresh()
}

// selectedTagCover returns the decoded cover of the selected entry,
// decoding once if the entry predates the per-entry bitmap cache.
func (a *app) selectedTagCover() *ui.Bitmap {
	s := &a.tagStudio
	entry := s.activeEntry(s.selected)
	if entry == nil {
		return nil
	}
	if entry.cover == nil {
		entry.cover = tagCoverBitmap(entry.info.Cover)
	}
	return entry.cover
}

func (a *app) discardTagChanges() {
	s := &a.tagStudio
	s.confirmDiscard = false
	s.draft = s.original
	s.cover = a.selectedTagCover()
	s.trackText = numberText(s.original.TrackNumber, s.original.TrackTotal)
	s.discText = numberText(s.original.DiscNumber, s.original.DiscTotal)
	s.dirty = false
	action := s.pendingAction
	s.pendingAction = nil
	if action != nil {
		action()
	}
	a.refresh()
}

func (a *app) tagStudioDiscardModal(c *ui.Context, col colors) {
	s := &a.tagStudio
	a.modal(c, col, &s.confirmDiscard, "未保存的更改", 380, func(close func()) {
		ui.Row(c).Gap(10).AlignItems(ui.Start).Children(func() {
			ui.Icon(c, iconAlert).Size(20, 20).TextColor(col.gold)
			ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, "当前文件的标签还没有保存。").FontSize(14).TextColor(col.text).LineHeight(1.35)
				ui.Text(c, "继续操作会丢弃这些修改，此操作无法撤销。").FontSize(12).
					TextColor(col.muted).LineHeight(1.35)
			})
		})
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			secondaryButton(c, col, nil, "取消", func() {
				s.pendingAction = nil
				close()
			})
			dangerButton(c, col, "放弃更改", func() {
				close()
				a.discardTagChanges()
			})
		})
	})
}

func (a *app) requestTagSource(source string) {
	s := &a.tagStudio
	if s.source == source {
		return
	}
	a.requestTagStudioAction(func() {
		s.source = source
		s.selected = -1
		a.clearTagDraft()
		if len(s.activeEntries()) > 0 {
			s.selected = 0
			a.loadTagEntry(0, false)
		}
	})
}

func (a *app) clearTagDraft() {
	s := &a.tagStudio
	s.draft = tag.Info{}
	s.original = tag.Info{}
	s.cover = nil
	s.trackText = ""
	s.discText = ""
	s.loading = false
	s.loadedPath = ""
	s.dirty = false
}

func (a *app) loadTagEntry(index int, refresh bool) {
	s := &a.tagStudio
	entry := s.activeEntry(index)
	if entry == nil {
		return
	}
	if index == s.selected && entry.loaded && samePath(s.loadedPath, entry.path) {
		if refresh {
			a.refresh()
		}
		return
	}
	s.selected = index
	if entry.loaded {
		entry.err = ""
		a.showTagEntry(entry)
		if refresh {
			a.refresh()
		}
		return
	}
	if entry.err != "" && !refresh {
		a.clearTagDraft()
		return
	}
	entry.err = ""
	s.loading = true
	if s.loadedPath == "" {
		// 首次进入：没有可保持显示的旧表单，先清空并显示加载状态。
		s.draft = tag.Info{}
		s.original = tag.Info{}
		s.cover = nil
		s.trackText = ""
		s.discText = ""
		s.loadedPath = ""
		s.dirty = false
	}
	// 否则保留上一个文件的表单直到新标签读取完成，避免面板闪烁。
	if !tag.Supported(entry.path) {
		s.loading = false
		entry.err = tag.ErrUnsupported.Error()
		if refresh {
			a.refresh()
		}
		return
	}
	path := entry.path
	var info tag.Info
	a.async(&s.readSlot, 30*time.Second, func(ctx context.Context) error {
		var err error
		info, err = tag.Read(path)
		return err
	}, func(err error) {
		current := s.activeEntry(s.selected)
		if current == nil || !samePath(current.path, path) {
			return
		}
		s.loading = false
		if err != nil {
			current.loaded = false
			current.err = friendlyError(err)
			current.missing = errors.Is(err, os.ErrNotExist)
			if refresh {
				a.refresh()
			}
			return
		}
		current.loaded = true
		current.err = ""
		current.missing = false
		// 关联双向补齐：文件 COMMENT 里的 MID 填补空缺；数据库/搜索匹配
		// 已有的 MID 补进 info，下一次保存时随 COMMENT 帧落盘。
		if info.SongMID == "" && current.songMID != "" {
			info.SongMID = current.songMID
		}
		if info.AlbumMID == "" && current.albumMID != "" {
			info.AlbumMID = current.albumMID
		}
		if current.songMID == "" && info.SongMID != "" {
			current.songMID = info.SongMID
		}
		if current.albumMID == "" && info.AlbumMID != "" {
			current.albumMID = info.AlbumMID
		}
		current.info = info
		current.size = fsutil.FileSize(path)
		current.cover = tagCoverBitmap(info.Cover)
		a.showTagEntry(current)
	})
	if refresh {
		a.refresh()
	}
}

// showTagEntry fills the editor from an entry whose tags are already in
// memory, so switching back to a file does not read it again or decode its
// cover again.
func (a *app) showTagEntry(entry *tagEntry) {
	s := &a.tagStudio
	s.loading = false
	s.draft = entry.info
	s.original = entry.info
	s.loadedPath = entry.path
	s.trackText = numberText(entry.info.TrackNumber, entry.info.TrackTotal)
	s.discText = numberText(entry.info.DiscNumber, entry.info.DiscTotal)
	s.cover = entry.cover
	if s.cover == nil && len(entry.info.Cover) > 0 {
		// The entry was saved with a new cover after its bitmap was
		// invalidated; decode the embedded art once, now.
		s.cover = tagCoverBitmap(entry.info.Cover)
		entry.cover = s.cover
	}
	s.dirty = false
}

// tagStudioNeighbor returns the next (dir 1) or previous (dir -1) entry in
// the current display order. Navigation follows the filtered, sorted list;
// a selection outside it moves to the nearest end.
func (a *app) tagStudioNeighbor(dir int) (int, bool) {
	s := &a.tagStudio
	order := a.matchedTagEntries()
	if len(order) == 0 {
		return 0, false
	}
	pos := -1
	for i, index := range order {
		if index == s.selected {
			pos = i
			break
		}
	}
	if pos < 0 {
		if dir > 0 {
			return order[0], true
		}
		return order[len(order)-1], true
	}
	next := pos + dir
	if next < 0 || next >= len(order) {
		return 0, false
	}
	return order[next], true
}

// tagStudioStep moves to the neighbouring entry, asking before an unsaved
// draft is dropped.
func (a *app) tagStudioStep(dir int) {
	a.requestTagStudioAction(func() {
		if index, ok := a.tagStudioNeighbor(dir); ok {
			a.loadTagEntry(index, false)
		}
	})
}

func (a *app) revertTagStudio() {
	s := &a.tagStudio
	if s.saving {
		return
	}
	s.draft = s.original
	s.cover = a.selectedTagCover()
	s.trackText = numberText(s.draft.TrackNumber, s.draft.TrackTotal)
	s.discText = numberText(s.draft.DiscNumber, s.draft.DiscTotal)
	s.dirty = false
	a.refresh()
}

func (a *app) saveTagStudio() {
	s := &a.tagStudio
	if s.batchBusy {
		a.setToast("批量补全进行中，请稍候")
		return
	}
	if s.saving {
		a.setToast("正在保存，请稍候")
		return
	}
	if s.loading || s.selected < 0 || !s.dirty {
		return
	}
	entry := s.activeEntry(s.selected)
	if entry == nil || !entry.loaded {
		return
	}
	oldPath := entry.path
	info := s.draft
	trackNum, trackTotal, hasTrackTotal := numberPairText(s.trackText)
	discNum, discTotal, hasDiscTotal := numberPairText(s.discText)
	info.TrackNumber = trackNum
	info.DiscNumber = discNum
	if hasTrackTotal {
		info.TrackTotal = trackTotal
	}
	if hasDiscTotal {
		info.DiscTotal = discTotal
	}
	target := a.tagStudioTargetPath(oldPath, info)
	s.saving = true
	a.refresh()

	finalPath := oldPath
	var size int64
	var renameErr error
	a.async(&s.saveSlot, 90*time.Second, func(ctx context.Context) error {
		if err := tag.Write(oldPath, info); err != nil {
			return err
		}
		if !samePath(target, oldPath) {
			if err := renameTagFile(oldPath, target); err != nil {
				renameErr = err
			} else {
				finalPath = target
			}
		}
		size = fsutil.FileSize(finalPath)
		return nil
	}, func(err error) {
		s.saving = false
		if err != nil {
			a.toast = "保存失败：" + friendlyError(err)
			a.refresh()
			return
		}
		a.finishTagSave(oldPath, finalPath, info, size, renameErr)
	})
}

func (a *app) finishTagSave(oldPath, newPath string, info tag.Info, size int64, renameErr error) {
	s := &a.tagStudio
	selectedPath := ""
	if current := s.activeEntry(s.selected); current != nil {
		selectedPath = current.path
	}
	for _, entries := range [][]tagEntry{s.historyEntries, s.fileEntries} {
		for i := range entries {
			if !samePath(entries[i].path, oldPath) {
				continue
			}
			entries[i].path = newPath
			entries[i].title = info.Title
			entries[i].artist = info.Artist
			entries[i].album = info.Album
			if !bytes.Equal(entries[i].info.Cover, info.Cover) {
				entries[i].cover = nil // decoded again when the entry is revisited
			}
			entries[i].info = info
			entries[i].loaded = true
			entries[i].err = ""
			entries[i].missing = false
			if size > 0 {
				entries[i].size = size
			}
		}
	}
	if selectedPath != "" && samePath(selectedPath, oldPath) {
		s.original = info
		s.draft = info
		s.loadedPath = newPath
		s.trackText = numberText(info.TrackNumber, info.TrackTotal)
		s.discText = numberText(info.DiscNumber, info.DiscTotal)
		s.dirty = false
	}
	a.afterTagSave(oldPath, newPath, info, size)
	switch {
	case renameErr != nil:
		a.toast = "标签已保存，但重命名失败：" + friendlyError(renameErr)
	case !samePath(oldPath, newPath):
		a.toast = "标签已保存，文件已重命名"
	default:
		a.toast = "标签已保存"
	}
	a.refresh()
}

// afterTagSave keeps the history page, the queue and the database in step
// with the file's new path and metadata.
func (a *app) afterTagSave(oldPath, newPath string, info tag.Info, size int64) {
	updateSize := size
	if updateSize <= 0 {
		updateSize = fsutil.FileSize(newPath)
	}
	for i := range a.history.items {
		record := &a.history.items[i]
		if !samePath(record.FilePath, oldPath) {
			continue
		}
		if updateSize <= 0 {
			updateSize = record.FileSize
		}
		if updateSize > 0 && record.FileSize != updateSize {
			a.history.bytes += updateSize - record.FileSize
			record.FileSize = updateSize
		}
		record.FilePath = newPath
		record.Title = info.Title
		record.Artist = info.Artist
		record.Album = info.Album
	}
	for i := range a.queue.items {
		item := &a.queue.items[i]
		if !samePath(item.FilePath, oldPath) {
			continue
		}
		item.FilePath = newPath
		if updateSize > 0 {
			item.FileSize = updateSize
		}
		a.persistQueueItem(*item)
	}
	store := a.history.store
	if store == nil {
		return
	}
	a.dispatchDisk(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := store.UpdateFile(ctx, oldPath, newPath, info.Title, info.Artist, info.Album, updateSize)
		if err != nil {
			log.Printf("update history after tag save: %v", err)
		}
	})
}

// tagStudioTargetPath expands the naming template for a tag save. Files
// inside the download directory move as a fresh download would; files
// elsewhere stay in their folder and only take the template's base name.
func (a *app) tagStudioTargetPath(oldPath string, info tag.Info) string {
	rendered, err := fsutil.RenderNameTemplate(a.nameTemplate, info.Artist, info.Album, info.Title)
	if err != nil {
		return oldPath
	}
	ext := filepath.Ext(oldPath)
	target := ""
	if strings.TrimSpace(a.downloadDir) != "" {
		if rel, err := filepath.Rel(a.downloadDir, oldPath); err == nil &&
			rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			if path, err := fsutil.SafePath(a.downloadDir, rendered+ext); err == nil {
				target = path
			}
		}
	}
	if target == "" {
		target = filepath.Join(filepath.Dir(oldPath), filepath.Base(rendered)+ext)
	}
	if samePath(target, oldPath) {
		return oldPath
	}
	return target
}

func renameTagFile(oldPath, newPath string) error {
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return err
	}
	if target, err := os.Stat(newPath); err == nil {
		if source, err := os.Stat(oldPath); err != nil || !os.SameFile(source, target) {
			return fmt.Errorf("目标文件已存在：%s", filepath.Base(newPath))
		}
	}
	return os.Rename(oldPath, newPath)
}

func numberString(value int) string {
	if value <= 0 {
		return ""
	}
	return strconv.Itoa(value)
}

func tagCoverBitmap(data []byte) *ui.Bitmap {
	if len(data) == 0 {
		return nil
	}
	bitmap, _ := ui.DecodeBitmap(data)
	return bitmap
}

// numberText formats a track or disc field: "3" alone, or "3/12" when the
// file carries a total.
func numberText(value, total int) string {
	if value > 0 && total > 0 {
		return strconv.Itoa(value) + "/" + strconv.Itoa(total)
	}
	return numberString(value)
}

// numberPairText reads a track or disc field like "3" or "3/12". hasTotal
// reports a usable "/total" part, so a plain number keeps the total already
// in the draft instead of clearing it.
func numberPairText(value string) (number, total int, hasTotal bool) {
	value = strings.TrimSpace(value)
	slash := strings.IndexByte(value, '/')
	if slash < 0 {
		return leadingInt(value), 0, false
	}
	total = leadingInt(strings.TrimSpace(value[slash+1:]))
	return leadingInt(strings.TrimSpace(value[:slash])), total, total > 0
}

func leadingInt(value string) int {
	value = strings.TrimSpace(value)
	number := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			break
		}
		number = number*10 + int(r-'0')
	}
	return number
}

func (a *app) pickTagStudioCover() {
	win := a.win
	if win == nil {
		return
	}
	entry := a.tagStudio.activeEntry(a.tagStudio.selected)
	if entry == nil {
		return
	}
	target := entry.path
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:  win,
			Title:   "选择封面图片",
			Filters: []mygo.FileFilter{{Name: "图片", Extensions: []string{"png", "jpg", "jpeg"}}},
		})
		if err != nil {
			a.update(func() { a.toast = "打开图片选择器失败：" + err.Error() })
			return
		}
		if len(paths) == 0 {
			return
		}
		data, err := os.ReadFile(paths[0])
		if err != nil {
			a.update(func() { a.toast = "读取图片失败：" + friendlyError(err) })
			return
		}
		if len(data) > maxTagCoverBytes {
			a.update(func() { a.toast = "图片过大，请选择 8 MB 以内的封面" })
			return
		}
		mime := tag.ImageMIME(data)
		if mime == "" {
			a.update(func() { a.toast = "仅支持 PNG / JPEG 图片" })
			return
		}
		bitmap, err := ui.DecodeBitmap(data)
		if err != nil {
			a.update(func() { a.toast = "图片无法解析：" + friendlyError(err) })
			return
		}
		a.update(func() {
			s := &a.tagStudio
			current := s.activeEntry(s.selected)
			if current == nil || !samePath(current.path, target) {
				return
			}
			s.draft.Cover = data
			s.draft.CoverMIME = mime
			s.cover = bitmap
			s.dirty = true
			a.refresh()
		})
	}()
}

func (a *app) removeTagStudioCover() {
	s := &a.tagStudio
	s.draft.Cover = nil
	s.draft.CoverMIME = ""
	s.cover = nil
	s.dirty = true
	a.refresh()
}

package main

// Tag Studio 与下载历史的同步（列表重建、记录增删改的联动），以及依赖
// QQ 音乐在线查询的动作：补全、获取歌词、搜索匹配。

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/history"
	"github.com/rzh0504/qmdr/internal/lrc"
	"github.com/rzh0504/qmdr/internal/qqmusic"
)

// tagEntryFromRecord builds the list entry a history record describes.
func tagEntryFromRecord(record history.Record) tagEntry {
	return tagEntry{
		path:      record.FilePath,
		historyID: record.ID,
		songMID:   record.SongMID,
		albumMID:  record.AlbumMID,
		title:     record.Title,
		artist:    record.Artist,
		album:     record.Album,
		size:      record.FileSize,
	}
}

func (a *app) rebuildTagHistory(keepSelection bool) {
	s := &a.tagStudio
	keepPath := ""
	if keepSelection {
		if entry := s.activeEntry(s.selected); entry != nil && s.source == "history" {
			keepPath = entry.path
		}
	}
	previous := make(map[string]tagEntry, len(s.historyEntries))
	for _, entry := range s.historyEntries {
		previous[pathKey(entry.path)] = entry
	}
	entries := make([]tagEntry, 0, len(a.history.items))
	seen := make(map[string]bool, len(a.history.items))
	for _, record := range a.history.items {
		if strings.TrimSpace(record.FilePath) == "" {
			continue
		}
		key := pathKey(record.FilePath)
		if seen[key] {
			continue
		}
		seen[key] = true
		entry := tagEntryFromRecord(record)
		if old, ok := previous[key]; ok && old.loaded {
			entry.info = old.info
			entry.loaded = true
			entry.size = old.size
			entry.cover = old.cover
		}
		entries = append(entries, entry)
	}
	s.historyEntries = entries
	s.selected = -1
	if keepPath != "" {
		for i := range entries {
			if samePath(entries[i].path, keepPath) {
				s.selected = i
				break
			}
		}
	}
}

// syncTagStudioHistory keeps an open Tag Studio list in step with a fresh
// download record: merged stale rows disappear and replaced rows take the
// new path. The next open reads the same data from the history page.
func (a *app) syncTagStudioHistory(record history.Record, outcome history.SaveOutcome, removed []int64) {
	s := &a.tagStudio
	if !s.historyReady {
		return
	}
	for _, removedID := range removed {
		a.removeTagEntriesByHistoryID(removedID)
	}
	switch outcome {
	case history.DownloadUpdated:
		a.updateTagEntriesForRecord(record)
	case history.DownloadInserted:
		a.prependTagHistoryEntry(record)
	}
}

func (a *app) removeTagEntriesByHistoryID(id int64) {
	s := &a.tagStudio
	s.historyEntries = a.removeTagEntry(s.historyEntries, s.source == "history", id)
	s.fileEntries = a.removeTagEntry(s.fileEntries, s.source == "files", id)
}

func (a *app) removeTagEntry(entries []tagEntry, active bool, id int64) []tagEntry {
	s := &a.tagStudio
	for i := range entries {
		if entries[i].historyID != id {
			continue
		}
		if active {
			if s.selected == i {
				a.clearTagDraft()
				s.selected = -1
			} else if s.selected > i {
				s.selected--
			}
		}
		return append(entries[:i], entries[i+1:]...)
	}
	return entries
}

func (a *app) updateTagEntriesForRecord(record history.Record) {
	s := &a.tagStudio
	a.updateTagEntryList(&s.historyEntries, s.source == "history", record)
	a.updateTagEntryList(&s.fileEntries, s.source == "files", record)
}

func (a *app) updateTagEntryList(entries *[]tagEntry, active bool, record history.Record) {
	s := &a.tagStudio
	for i := range *entries {
		entry := &(*entries)[i]
		if entry.historyID != record.ID {
			continue
		}
		entry.path = record.FilePath
		entry.songMID = record.SongMID
		entry.albumMID = record.AlbumMID
		entry.title = record.Title
		entry.artist = record.Artist
		entry.album = record.Album
		entry.size = record.FileSize
		entry.loaded = false
		entry.err = ""
		entry.missing = false
		if active && s.selected == i && !s.dirty {
			a.clearTagDraft()
			a.loadTagEntry(i, false)
		}
		return
	}
}

func (a *app) prependTagHistoryEntry(record history.Record) {
	s := &a.tagStudio
	entry := tagEntryFromRecord(record)
	for i := range s.historyEntries {
		if s.historyEntries[i].historyID == record.ID {
			s.historyEntries[i] = entry
			return
		}
	}
	s.historyEntries = append(s.historyEntries, tagEntry{})
	copy(s.historyEntries[1:], s.historyEntries[:len(s.historyEntries)-1])
	s.historyEntries[0] = entry
	if s.source == "history" && s.selected >= 0 {
		s.selected++
	}
}

// removeTagHistoryEntryByID forgets a download record that the history page
// deleted; files opened from disk stay in the files source.
func (a *app) removeTagHistoryEntryByID(id int64) {
	s := &a.tagStudio
	s.historyEntries = a.removeTagEntry(s.historyEntries, s.source == "history", id)
}

func (a *app) clearTagHistoryEntries() {
	s := &a.tagStudio
	s.historyEntries = nil
	if s.source == "history" {
		s.selected = -1
		a.clearTagDraft()
	}
}

// tagStudioDraftLostByHistoryDelete reports whether deleting history
// records would drop unsaved edits in the open Tag Studio draft.
func (a *app) tagStudioDraftLostByHistoryDelete(ids []int64) bool {
	s := &a.tagStudio
	if !s.dirty || s.source != "history" {
		return false
	}
	current := s.activeEntry(s.selected)
	if current == nil || current.historyID == 0 {
		return false
	}
	for _, id := range ids {
		if current.historyID == id {
			return true
		}
	}
	return false
}

func (a *app) retryTagHistory() {
	if a.history.loading {
		a.setToast("下载历史正在加载")
		return
	}
	a.requestTagStudioAction(func() {
		s := &a.tagStudio
		s.historyReady = false
		s.historyEntries = nil
		s.selected = -1
		a.clearTagDraft()
		a.retryHistory()
	})
}

func (a *app) refreshTagHistory() {
	if a.history.loading {
		a.setToast("下载历史正在加载")
		return
	}
	a.requestTagStudioAction(func() {
		s := &a.tagStudio
		a.rebuildTagHistory(true)
		if s.selected < 0 {
			a.clearTagDraft()
			if len(s.historyEntries) > 0 {
				s.selected = 0
				a.loadTagEntry(0, false)
			}
		}
	})
}

func (a *app) openTagStudioRecord(record history.Record) {
	action := func() {
		a.view = viewTagStudio
		s := &a.tagStudio
		s.source = "history"
		if !s.historyReady {
			s.historyReady = true
			a.rebuildTagHistory(false)
		}
		index := a.tagHistoryIndex(record.FilePath)
		if index < 0 {
			s.historyEntries = append(s.historyEntries, tagEntryFromRecord(record))
			index = len(s.historyEntries) - 1
		}
		current := s.activeEntry(s.selected)
		if current == nil || !samePath(current.path, record.FilePath) {
			s.selected = -1
			a.clearTagDraft()
			a.loadTagEntry(index, false)
		}
	}
	if a.view == viewTagStudio && a.tagStudio.source == "history" {
		if entry := a.tagStudio.activeEntry(a.tagStudio.selected); entry != nil &&
			samePath(entry.path, record.FilePath) {
			return
		}
	}
	a.requestTagStudioAction(action)
}

func (a *app) tagHistoryIndex(path string) int {
	for i := range a.tagStudio.historyEntries {
		if samePath(a.tagStudio.historyEntries[i].path, path) {
			return i
		}
	}
	return -1
}

func (a *app) fillTagStudioMeta() {
	s := &a.tagStudio
	entry := s.activeEntry(s.selected)
	if entry == nil || s.metaBusy {
		return
	}
	if !a.api.HasCredential() {
		a.setToast("请先登录 QQ 音乐")
		return
	}
	if entry.songMID == "" {
		a.setToast("该文件没有关联 QQ 音乐歌曲 ID")
		return
	}
	songMID := entry.songMID
	target := entry.path
	s.metaBusy = true
	a.refresh()
	var meta qqmusic.SongMeta
	var credits qqmusic.SongCredits
	a.async(&s.metaSlot, 30*time.Second, func(ctx context.Context) error {
		var err error
		meta, err = a.api.SongMeta(ctx, songMID)
		if err != nil {
			return err
		}
		credits, err = a.api.SongCredits(ctx, songMID)
		if err != nil {
			log.Printf("load song credits: %v", err)
			credits = qqmusic.SongCredits{}
		}
		return nil
	}, func(err error) {
		s.metaBusy = false
		current := s.activeEntry(s.selected)
		if current == nil || !samePath(current.path, target) {
			a.refresh()
			return
		}
		if err != nil {
			a.toast = "补全失败：" + friendlyError(err)
			a.refresh()
			return
		}
		filled := 0
		fill := func(dst *string, value string) {
			if strings.TrimSpace(*dst) == "" && strings.TrimSpace(value) != "" {
				*dst = value
				filled++
			}
		}
		fill(&s.draft.Title, current.title)
		fill(&s.draft.Artist, current.artist)
		fill(&s.draft.Album, current.album)
		fill(&s.draft.Date, meta.ReleaseDate)
		fill(&s.draft.Genre, meta.Genre)
		fill(&s.draft.Language, meta.Language)
		fill(&s.draft.Composer, credits.Composer)
		fill(&s.draft.Lyricist, credits.Lyricist)
		if filled > 0 {
			s.dirty = true
			a.toast = fmt.Sprintf("已补全 %d 项信息", filled)
		} else {
			a.toast = "没有可补全的信息"
		}
		a.refresh()
	})
}

// requestTagLyrics asks before a fetch would replace lyrics the user edited
// in the draft; fetching over untouched lyrics just runs.
func (a *app) requestTagLyrics() {
	s := &a.tagStudio
	entry := s.activeEntry(s.selected)
	if entry == nil {
		return
	}
	if entry.songMID == "" {
		a.setToast("该文件没有关联 QQ 音乐歌曲 ID")
		return
	}
	if strings.TrimSpace(s.draft.Lyrics) != "" && s.draft.Lyrics != s.original.Lyrics {
		s.confirmLyricsOverwrite = true
		a.refresh()
		return
	}
	a.fetchTagStudioLyrics()
}

func (a *app) tagStudioLyricsModal(c *ui.Context, col colors) {
	a.modal(c, col, &a.tagStudio.confirmLyricsOverwrite, "覆盖已修改的歌词", 380, func(close func()) {
		ui.Row(c).Gap(10).AlignItems(ui.Start).Children(func() {
			ui.Icon(c, iconAlert).Size(20, 20).TextColor(col.gold)
			ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, "当前歌词已被手动修改。").FontSize(14).TextColor(col.text).LineHeight(1.35)
				ui.Text(c, "获取歌词会覆盖这些修改，且无法恢复。").FontSize(12).
					TextColor(col.muted).LineHeight(1.35)
			})
		})
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			secondaryButton(c, col, nil, "取消", close)
			dangerButton(c, col, "覆盖获取", func() {
				close()
				a.fetchTagStudioLyrics()
			})
		})
	})
}

func (a *app) fetchTagStudioLyrics() {
	s := &a.tagStudio
	entry := s.activeEntry(s.selected)
	if entry == nil || s.lyricBusy {
		return
	}
	if !a.api.HasCredential() {
		a.setToast("请先登录 QQ 音乐")
		return
	}
	if entry.songMID == "" {
		a.setToast("该文件没有关联 QQ 音乐歌曲 ID")
		return
	}
	songMID := entry.songMID
	target := entry.path
	withTranslation := a.lyricsTrans
	s.lyricBusy = true
	a.refresh()
	var text string
	a.async(&s.lyricSlot, 30*time.Second, func(ctx context.Context) error {
		lyrics, err := a.api.Lyric(ctx, songMID)
		if err != nil {
			return err
		}
		text = lyrics.LRC
		if withTranslation {
			text = lrc.Merge(text, lyrics.Translation)
		}
		return nil
	}, func(err error) {
		s.lyricBusy = false
		current := s.activeEntry(s.selected)
		if current == nil || !samePath(current.path, target) {
			a.refresh()
			return
		}
		switch {
		case err != nil:
			a.toast = "获取歌词失败：" + friendlyError(err)
		case strings.TrimSpace(text) == "":
			a.toast = "没有获取到歌词"
		default:
			s.draft.Lyrics = text
			s.dirty = true
			a.toast = "歌词已更新"
		}
		a.refresh()
	})
}

// openTagMatch starts a song search for the selected entry, so files the
// download history does not know can still use the QQ Music lookups. The
// query is built from the file's own tags or name and the search fires
// right away; the field stays editable for a re-search.
func (a *app) openTagMatch() {
	s := &a.tagStudio
	entry := s.activeEntry(s.selected)
	if entry == nil {
		return
	}
	s.matchFor = entry.path
	s.matchQuery = a.tagMatchQuery(entry)
	s.matchResults = nil
	s.matchErr = ""
	s.matchOpen = true
	a.refresh()
	if strings.TrimSpace(s.matchQuery) != "" {
		a.searchTagMatch()
	}
}

// tagMatchNoise strips bracketed noise like [HQ] or 【无损】 from a
// filename-derived query.
var tagMatchNoise = regexp.MustCompile(`[\[【][^\]】]*[\]】]`)

// tagMatchQuery builds the search query for an entry from its own tags,
// falling back to its file name.
func (a *app) tagMatchQuery(entry *tagEntry) string {
	title := tagMatchNoise.ReplaceAllString(strings.TrimSpace(entry.displayTitle()), "")
	title = strings.Join(strings.Fields(title), " ")
	artist := tagMatchNoise.ReplaceAllString(strings.TrimSpace(entry.displayArtist()), "")
	artist = strings.Join(strings.Fields(artist), " ")
	switch {
	case title != "" && artist != "":
		return title + " " + artist
	case title != "":
		return title
	}
	return artist
}

func (a *app) searchTagMatch() {
	s := &a.tagStudio
	if s.matchLoading {
		return
	}
	query := strings.TrimSpace(s.matchQuery)
	if query == "" {
		a.setToast("请输入歌曲名或歌手")
		return
	}
	s.matchLoading = true
	a.refresh()
	var songs []qqmusic.Song
	a.async(&s.matchSlot, 30*time.Second, func(ctx context.Context) error {
		page, err := a.api.Search(ctx, query, "", 1, 20)
		if err != nil {
			return err
		}
		songs = page.Songs
		return nil
	}, func(err error) {
		s.matchLoading = false
		if err != nil {
			s.matchErr = friendlyError(err)
			s.matchResults = nil
			a.refresh()
			return
		}
		s.matchErr = ""
		s.matchResults = songs
		a.refresh()
	})
}

// bindTagMatch links the picked song to the entry the search started from,
// which is enough for the completion, lyric and cover lookups to work. The
// binding marks the draft dirty, so the next save persists it into the
// file's COMMENT frame.
func (a *app) bindTagMatch(song qqmusic.Song) {
	s := &a.tagStudio
	entry := s.activeEntry(s.selected)
	if entry == nil || !samePath(entry.path, s.matchFor) {
		s.matchOpen = false
		a.refresh()
		return
	}
	changed := entry.songMID != song.Mid || entry.albumMID != song.AlbumMid
	entry.songMID = song.Mid
	entry.albumMID = song.AlbumMid
	if entry.loaded {
		entry.info.SongMID = song.Mid
		entry.info.AlbumMID = song.AlbumMid
		s.draft.SongMID = song.Mid
		s.draft.AlbumMID = song.AlbumMid
		if changed {
			s.dirty = true
		}
	}
	s.matchOpen = false
	a.setToast("已关联「" + song.Title + "」，保存后关联写入文件")
}

func (a *app) tagMatchModal(c *ui.Context, col colors) {
	s := &a.tagStudio
	a.modal(c, col, &s.matchOpen, "搜索匹配歌曲", 460, func(close func()) {
		ui.Column(c).FillWidth().Gap(12).Children(func() {
			ui.Text(c, "已按文件信息自动搜索，结果不对可修改关键词重搜；点选关联后即可补全信息，保存时关联写入文件。").
				FontSize(12).TextColor(col.muted).LineHeight(1.35)
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				a.textField(c, col, &s.matchQuery, "歌曲名或歌手", 0, 30, nil, a.searchTagMatch)
				if s.matchLoading {
					ui.Spinner(c).Size(14, 14).Shrink(0)
				} else {
					secondaryButton(c, col, iconSearch, "搜索", a.searchTagMatch)
				}
			})
			switch {
			case s.matchErr != "":
				ui.Row(c).Gap(9).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconAlert).Size(14, 14).TextColor(col.red)
					ui.Text(c, s.matchErr).FontSize(12).TextColor(col.red).Grow(1).MaxLines(2)
				})
			case len(s.matchResults) == 0 && !s.matchLoading:
				ui.Text(c, "输入关键词开始搜索").FontSize(12).TextColor(col.faint)
			}
			if len(s.matchResults) > 0 {
				ui.Scroll(c).FillWidth().Height(264).Gap(1).Children(func() {
					for _, song := range s.matchResults {
						song := song
						row := ui.ButtonBase(c).FillWidth().Height(46).Radius(8).
							Padding(6, 8).Gap(9).AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(song.Title)
						if row.Hovered() {
							row.Background(col.sheetHover)
						}
						if row.Clicked() {
							close()
							a.bindTagMatch(song)
						}
						row.Children(func() {
							ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
								ui.Text(c, song.Title).FontSize(13).TextColor(col.text).SingleLine()
								detail := strings.Join(song.Artists, " / ")
								if song.Album != "" {
									if detail != "" {
										detail += " · "
									}
									detail += song.Album
								}
								ui.Text(c, detail).FontSize(11).TextColor(col.muted).SingleLine().Ellipsis("…")
							})
							if song.VIP {
								ui.Text(c, "VIP").FontSize(9).FontWeight(600).TextColor(col.gold).Shrink(0)
							}
						})
					}
				})
			}
		})
	})
}

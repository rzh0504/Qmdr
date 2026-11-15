package main

// Tag Studio 页面骨架与列表：视图入口、侧栏（来源切换/筛选/排序）、列表行、
// 面板分发，以及本地文件夹来源的文件管理。

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/tag"
)

// maxTagFolderFiles caps how many audio files one folder scan adds, so a
// huge tree cannot flood the list.
const maxTagFolderFiles = 500

// tagSortNatural is the list's own order: the history source shows the
// download order, the files source shows the order files were opened.
const tagSortNatural = "下载时间"

var tagSortOptions = []string{tagSortNatural, "标题", "艺人", "专辑"}

// tagSortPair is one entry's sort key with its index into the entry list.
type tagSortPair struct {
	index int
	key   string
}

func (a *app) tagStudioView(c *ui.Context, col colors) {
	s := &a.tagStudio
	if s.source == "" {
		s.source = "history"
	}
	if s.sortBy == "" {
		s.sortBy = tagSortNatural
	}
	if s.source == "history" && !s.historyReady && !a.history.loading {
		s.historyReady = true
		a.rebuildTagHistory(false)
		if s.selected < 0 && len(s.historyEntries) > 0 {
			s.selected = 0
			a.loadTagEntry(0, false)
		}
	}
	if s.batchMode {
		a.tagBatchView(c, col)
		return
	}
	c.OnShortcut(ui.Cmd, ui.KeyS, a.saveTagStudio)
	c.OnShortcut(ui.Alt, ui.KeyLeft, func() { a.tagStudioStep(-1) })
	c.OnShortcut(ui.Alt, ui.KeyRight, func() { a.tagStudioStep(1) })
	ui.Row(c).Fill().AlignItems(ui.Stretch).Gap(16).Children(func() {
		a.tagStudioSidebar(c, col)
		a.tagStudioPanel(c, col)
	})
}

func (a *app) tagStudioSidebar(c *ui.Context, col colors) {
	s := &a.tagStudio
	ui.Column(c).Width(264).FillHeight().MinHeight(0).Gap(12).Children(func() {
		ui.Row(c).Height(30).FillWidth().AlignItems(ui.Center).Gap(4).Children(func() {
			a.tagStudioSourceSwitch(c, col)
			ui.Spacer(c)
			if s.source == "history" {
				iconButton(c, col, iconRefresh, "刷新下载历史", a.refreshTagHistory)
			}
		})
		searchBox(c, col, &s.filter, "筛选标题、艺人或路径", nil, func() {
			s.filter = ""
			a.refresh()
		})
		ui.Row(c).FillWidth().Height(34).Gap(7).Padding(0, 11).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "排序").FontSize(12).TextColor(col.faint).Shrink(0)
			selectControl(c, col, &s.sortBy, tagSortOptions, 120, col.control, a.refresh)
		})
		matched := a.matchedTagEntries()
		listPanel(c, col, "tag-studio-"+s.source, func() {
			if len(matched) == 0 {
				if len(s.activeEntries()) == 0 {
					if s.source == "files" {
						ui.Column(c).FillWidth().AlignItems(ui.Center).
							Padding(56, 0, 0, 0).Children(func() {
							a.tagFolderInvite(c, col)
						})
					} else {
						ui.Text(c, "暂无文件").FontSize(12).TextColor(col.faint).Padding(8, 10)
					}
				} else {
					ui.Text(c, "没有匹配的文件").FontSize(12).TextColor(col.faint).Padding(8, 10)
				}
				return
			}
			for _, index := range matched {
				a.tagEntryRow(c, col, index)
			}
		})
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			if len(matched) == len(s.activeEntries()) {
				ui.Textf(c, "共 %d 个文件", len(matched)).FontSize(11).
					TextColor(col.faint).Padding(0, 4)
			} else {
				ui.Textf(c, "匹配 %d / 共 %d 个文件", len(matched), len(s.activeEntries())).FontSize(11).
					TextColor(col.faint).Padding(0, 4)
			}
			if s.source == "files" {
				ui.Spacer(c)
				ghostButton(c, col, iconFolder, "选择文件夹", a.pickTagFolder)
			}
		})
	})
}

// matchedTagEntries returns the indices of the active entries that pass the
// current filter, in the display order, so the list, its count and the
// prev/next navigation all agree and the match runs once per frame.
func (a *app) matchedTagEntries() []int {
	s := &a.tagStudio
	entries := s.activeEntries()
	var indices []int
	if strings.TrimSpace(s.filter) == "" {
		indices = make([]int, len(entries))
		for i := range indices {
			indices[i] = i
		}
	} else {
		indices = make([]int, 0, len(entries))
		for i := range entries {
			if a.tagEntryMatches(i) {
				indices = append(indices, i)
			}
		}
	}
	return a.sortedTagIndices(indices)
}

// sortedTagIndices orders the indices by the sort the sidebar picked: the
// natural list order, or an ascending field sort that keeps the natural
// order for equal keys.
func (a *app) sortedTagIndices(indices []int) []int {
	s := &a.tagStudio
	if s.sortBy == "" || s.sortBy == tagSortNatural {
		return indices
	}
	entries := s.activeEntries()
	pairs := make([]tagSortPair, len(indices))
	for i, index := range indices {
		key := ""
		switch s.sortBy {
		case "标题":
			key = entries[index].displayTitle()
		case "艺人":
			key = entries[index].displayArtist()
		case "专辑":
			key = entries[index].displayAlbum()
		}
		pairs[i] = tagSortPair{index: index, key: strings.ToLower(key)}
	}
	sort.SliceStable(pairs, func(x, y int) bool {
		return pairs[x].key < pairs[y].key
	})
	for i, pair := range pairs {
		indices[i] = pair.index
	}
	return indices
}

// tagStudioSourceSwitch changes sources through a request so a dirty draft
// is not dropped without confirmation. The highlight is a thumb painted in
// the track's Draw phase (under the labels) that eases to the active segment.
func (a *app) tagStudioSourceSwitch(c *ui.Context, col colors) {
	s := &a.tagStudio
	segW := float32(78)
	track := ui.Row(c).Height(30).Padding(2).Radius(8).AlignItems(ui.Center).Shrink(0)
	pos := track.Animate("source", b2f(s.source == "files"), 160*time.Millisecond)
	track.Draw(func(p *ui.Painter, r ui.Rect) {
		chip := smoothRectPath(snapRect(p, r), 8)
		p.FillPath(&chip, col.chip)
		thumb := smoothRectPath(snapRect(p, ui.Rect{
			X: r.X + 2 + pos*segW, Y: r.Y + 2, W: segW, H: 26,
		}), 6)
		p.FillPath(&thumb, col.segThumb)
	})
	track.Children(func() {
		for _, option := range []struct {
			key   string
			label string
		}{
			{"history", "下载历史"},
			{"files", "本地文件夹"},
		} {
			option := option
			active := s.source == option.key
			seg := ui.ButtonBase(c).Width(segW).Height(26).Radius(6).Center().
				Cursor(ui.CursorPointer).Label(option.label)
			if seg.Clicked() {
				a.requestTagSource(option.key)
			}
			fg := col.muted
			if active {
				fg = col.text
			}
			seg.Children(func() {
				ui.Text(c, option.label).FontSize(12).TextColor(fg).SingleLine()
			})
		}
	})
}

func (a *app) tagEntryRow(c *ui.Context, col colors, index int) {
	s := &a.tagStudio
	entry := &s.activeEntries()[index]
	selected := index == s.selected
	row := ui.ButtonBase(c).FillWidth().Height(54).Padding(6, 8).Gap(9).Radius(8).
		AlignItems(ui.Center).Label(entry.displayTitle())
	if selected {
		row.Background(col.rowSelected)
		row = chosen(row, col)
	} else if row.Hovered() {
		row.Background(col.sheetHover)
	}
	if row.Clicked() && index != s.selected {
		index := index
		a.requestTagStudioAction(func() { a.loadTagEntry(index, false) })
	}
	row.Children(func() {
		switch {
		case entry.cover != nil:
			// The entry is loaded: show the cover embedded in the file, the
			// same art the editor panel shows.
			thumb := ui.Box(c).Size(34, 34).Radius(4).Background(col.cardAlt).
				Border(1, col.border).Clip()
			thumb.Children(func() {
				ui.Image(c, entry.cover).Size(34, 34).Fit(ui.Cover)
			})
		case entry.albumMID != "":
			a.cover(c, col, entry.albumMID, 34)
		default:
			box := ui.Box(c).Size(34, 34).Radius(4).Background(col.cardAlt).
				Border(1, col.border).Clip()
			box.Children(func() {
				ui.Row(c).Fill().Center().Children(func() {
					ui.Icon(c, iconMusic).Size(15, 15).TextColor(col.muted)
				})
			})
		}
		ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, entry.displayTitle()).FontSize(13).TextColor(col.text).
					SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
				if selected && s.dirty {
					ui.Box(c).Size(6, 6).Radius(3).Background(col.accent).Shrink(0)
				}
			})
			detail := entry.displayArtist()
			if album := entry.displayAlbum(); album != "" {
				if detail != "" {
					detail += " · "
				}
				detail += album
			}
			if detail == "" {
				detail = filepath.Dir(entry.path)
			}
			ui.Text(c, detail).FontSize(11).TextColor(col.muted).
				SingleLine().Ellipsis("…")
		})
		if entry.missing {
			ui.Box(c).Size(6, 6).Radius(3).Background(col.red).Shrink(0)
		}
		ui.Text(c, entry.formatLabel()).FontSize(10).TextColor(col.faint).Shrink(0)
	})
}

func (a *app) tagStudioPanel(c *ui.Context, col colors) {
	s := &a.tagStudio
	ui.Column(c).Grow(1).MinWidth(0).FillHeight().MinHeight(0).Children(func() {
		if s.source == "history" && !s.historyReady && a.history.loading {
			loadingState(c, col, "正在加载下载历史…")
			return
		}
		entries := s.activeEntries()
		if len(entries) == 0 {
			a.tagStudioEmptyState(c, col)
			return
		}
		if s.selected < 0 || s.selected >= len(entries) {
			emptyState(c, col, iconTag, "选择一个文件开始编辑",
				"从左侧列表选择文件，或打开本地的 MP3 / FLAC 文件。")
			return
		}
		if s.loading && s.loadedPath == "" {
			loadingState(c, col, "正在读取标签…")
			return
		}
		entry := &entries[s.selected]
		if entry.err != "" && !entry.loaded {
			a.tagStudioReadError(c, col, entry)
			return
		}
		a.tagStudioForm(c, col, entry)
	})
}

func (a *app) tagStudioEmptyState(c *ui.Context, col colors) {
	s := &a.tagStudio
	title, hint := "下载历史为空", "下载过的歌曲会出现在这里。"
	if s.source == "files" {
		title, hint = "未选择本地文件夹", "选择一个文件夹，扫描其中的 MP3 / FLAC 添加到列表。"
	} else if a.history.err != "" {
		title, hint = "下载历史不可用", a.history.err
	}
	ui.Column(c).Fill().Center().Children(func() {
		ui.Column(c).Width(420).Children(func() {
			emptyState(c, col, iconTag, title, hint)
			if s.source == "history" {
				ui.Row(c).Gap(8).Justify(ui.Center).Children(func() {
					secondaryButton(c, col, iconRefresh, "重新加载", a.retryTagHistory)
				})
			}
		})
	})
}

func (a *app) tagStudioReadError(c *ui.Context, col colors, entry *tagEntry) {
	message := entry.err
	if entry.missing {
		message = "文件不存在或已被移动"
	}
	softCard(c, col, func() {
		ui.Row(c).Gap(9).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconAlert).Size(15, 15).TextColor(col.red)
			ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
				ui.Text(c, "无法读取标签："+message).FontSize(12).TextColor(col.red).MaxLines(3)
				ui.Text(c, entry.path).FontSize(11).TextColor(col.muted).
					SingleLine().Ellipsis("…").Tooltip(entry.path)
				if entry.missing && a.tagStudio.source == "history" {
					ui.Text(c, "可在下载历史中删除这条记录。").FontSize(11).TextColor(col.faint)
				}
			})
			secondaryButton(c, col, iconRefresh, "重试", func() {
				a.loadTagEntry(a.tagStudio.selected, true)
			})
			if a.tagStudio.source == "files" {
				index := a.tagStudio.selected
				secondaryButton(c, col, iconTrash, "从列表移除", func() {
					a.removeTagFile(index)
				})
			}
		})
	})
}

func (a *app) tagEntryMatches(index int) bool {
	s := &a.tagStudio
	query := strings.ToLower(strings.TrimSpace(s.filter))
	if query == "" {
		return true
	}
	entry := s.activeEntry(index)
	if entry == nil {
		return false
	}
	haystack := strings.ToLower(strings.Join([]string{
		entry.displayTitle(), entry.displayArtist(), entry.displayAlbum(), entry.path,
	}, " "))
	return strings.Contains(haystack, query)
}

// pickTagFolder opens a folder dialog and adds every MP3 / FLAC inside it,
// scanned recursively, to the local files list.
func (a *app) pickTagFolder() {
	win := a.win
	if win == nil {
		return
	}
	go func() {
		dirs, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:    win,
			Title:     "选择本地文件夹",
			Directory: true,
		})
		if err != nil {
			a.update(func() { a.toast = "打开文件夹选择器失败：" + err.Error() })
			return
		}
		if len(dirs) == 0 {
			return
		}
		var paths []string
		truncated := false
		err = filepath.WalkDir(dirs[0], func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !tag.Supported(path) {
				return nil
			}
			if len(paths) >= maxTagFolderFiles {
				truncated = true
				return filepath.SkipAll
			}
			paths = append(paths, path)
			return nil
		})
		if err != nil {
			a.update(func() { a.toast = "扫描文件夹失败：" + friendlyError(err) })
			return
		}
		sort.Strings(paths)
		a.update(func() {
			a.addTagFiles(paths)
			if truncated {
				a.toast = "文件夹较大，仅加载前 " + strconv.Itoa(maxTagFolderFiles) + " 个音频文件"
			}
		})
	}()
}

// tagFolderInvite is the empty local folder list's centered call to action.
func (a *app) tagFolderInvite(c *ui.Context, col colors) {
	invite := ui.ButtonBase(c).Column().Width(200).Height(92).Radius(10).
		Background(col.cardAlt).Border(1, col.border).
		Justify(ui.Center).AlignItems(ui.Center).Gap(8).
		Cursor(ui.CursorPointer).Label("选择本地文件夹")
	if invite.Hovered() {
		invite.Border(1, col.accent)
	}
	if invite.Clicked() {
		a.pickTagFolder()
	}
	invite.Children(func() {
		ui.Icon(c, iconFolder).Size(26, 26).TextColor(col.muted)
		ui.Text(c, "选择本地文件夹").FontSize(12).TextColor(col.muted)
	})
}

func (a *app) addTagFiles(paths []string) {
	s := &a.tagStudio
	first := ""
	added := 0
	for _, path := range paths {
		if !tag.Supported(path) {
			continue
		}
		if a.tagFileIndex(path) >= 0 {
			continue
		}
		entry := a.tagEntryFromPath(path)
		s.fileEntries = append(s.fileEntries, entry)
		added++
		if first == "" {
			first = path
		}
	}
	if added == 0 {
		a.setToast("没有可添加的新文件（可能已在列表中）")
		return
	}
	a.toast = fmt.Sprintf("已添加 %d 个本地文件", added)
	target := first
	a.requestTagStudioAction(func() {
		s.source = "files"
		index := a.tagFileIndex(target)
		if index < 0 {
			return
		}
		s.selected = -1
		a.clearTagDraft()
		a.loadTagEntry(index, false)
	})
}

// tagEntryFromPath fills a file entry with what the download history knows
// about the path, so QQ Music lookups also work for files opened from disk.
func (a *app) tagEntryFromPath(path string) tagEntry {
	entry := tagEntry{path: path}
	for _, record := range a.history.items {
		if !samePath(record.FilePath, path) {
			continue
		}
		entry.historyID = record.ID
		entry.songMID = record.SongMID
		entry.albumMID = record.AlbumMID
		entry.title = record.Title
		entry.artist = record.Artist
		entry.album = record.Album
		entry.size = record.FileSize
		break
	}
	return entry
}

func (a *app) tagFileIndex(path string) int {
	for i := range a.tagStudio.fileEntries {
		if samePath(a.tagStudio.fileEntries[i].path, path) {
			return i
		}
	}
	return -1
}

func (a *app) removeTagFile(index int) {
	s := &a.tagStudio
	if index < 0 || index >= len(s.fileEntries) {
		return
	}
	remove := func() {
		s.fileEntries = append(s.fileEntries[:index], s.fileEntries[index+1:]...)
		if s.selected == index {
			a.clearTagDraft()
			s.selected = -1
			if len(s.fileEntries) > 0 {
				s.selected = min(index, len(s.fileEntries)-1)
				a.loadTagEntry(s.selected, false)
			}
			return
		}
		if s.selected > index {
			s.selected--
		}
	}
	if index == s.selected && s.dirty {
		a.requestTagStudioAction(remove)
		return
	}
	remove()
	a.refresh()
}

// activeEntries returns the list the current source shows.
func (s *tagStudioState) activeEntries() []tagEntry {
	if s.source == "files" {
		return s.fileEntries
	}
	return s.historyEntries
}

func (s *tagStudioState) activeEntry(index int) *tagEntry {
	entries := s.activeEntries()
	if index < 0 || index >= len(entries) {
		return nil
	}
	return &entries[index]
}

func (e *tagEntry) displayTitle() string {
	if e.loaded && strings.TrimSpace(e.info.Title) != "" {
		return e.info.Title
	}
	if strings.TrimSpace(e.title) != "" {
		return e.title
	}
	base := filepath.Base(e.path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func (e *tagEntry) displayArtist() string {
	if e.loaded && strings.TrimSpace(e.info.Artist) != "" {
		return e.info.Artist
	}
	return e.artist
}

func (e *tagEntry) displayAlbum() string {
	if e.loaded && strings.TrimSpace(e.info.Album) != "" {
		return e.info.Album
	}
	return e.album
}

func (e *tagEntry) formatLabel() string {
	ext := strings.TrimPrefix(strings.ToUpper(filepath.Ext(e.path)), ".")
	if ext == "" {
		return "未知格式"
	}
	return ext
}

func pathKey(path string) string {
	key := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(key)
	}
	return key
}

func samePath(a, b string) bool {
	return pathKey(a) == pathKey(b)
}

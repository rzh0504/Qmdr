package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/format"
	"github.com/rzh0504/qmdr/internal/fsutil"
	"github.com/rzh0504/qmdr/internal/history"
	"github.com/rzh0504/qmdr/internal/lrc"
	"github.com/rzh0504/qmdr/internal/qqmusic"
	"github.com/rzh0504/qmdr/internal/settings"
	"github.com/rzh0504/qmdr/internal/tag"
)

const maxTagCoverBytes = 8 << 20

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
			iconButton(c, col, iconFolder, "打开本地文件", a.pickTagFiles)
			if s.source == "history" {
				iconButton(c, col, iconRefresh, "刷新下载历史", a.refreshTagHistory)
			}
		})
		searchBox(c, col, &s.filter, "筛选标题、艺人或路径", nil, func() {
			s.filter = ""
			a.refresh()
		})
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(c, "排序").FontSize(12).TextColor(col.faint).Shrink(0)
			selectControl(c, col, &s.sortBy, tagSortOptions, 120, col.control, a.refresh)
			ui.Spacer(c)
		})
		matched := a.matchedTagEntries()
		listPanel(c, col, "tag-studio-"+s.source, func() {
			if len(matched) == 0 {
				if len(s.activeEntries()) == 0 {
					ui.Text(c, "暂无文件").FontSize(12).TextColor(col.faint).Padding(8, 10)
				} else {
					ui.Text(c, "没有匹配的文件").FontSize(12).TextColor(col.faint).Padding(8, 10)
				}
				return
			}
			for _, index := range matched {
				a.tagEntryRow(c, col, index)
			}
		})
		if len(matched) == len(s.activeEntries()) {
			ui.Textf(c, "共 %d 个文件", len(matched)).FontSize(11).
				TextColor(col.faint).Padding(0, 4)
		} else {
			ui.Textf(c, "匹配 %d / 共 %d 个文件", len(matched), len(s.activeEntries())).FontSize(11).
				TextColor(col.faint).Padding(0, 4)
		}
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
// is not dropped without confirmation.
func (a *app) tagStudioSourceSwitch(c *ui.Context, col colors) {
	s := &a.tagStudio
	track := ui.Row(c).Height(30).Padding(2).Radius(8).AlignItems(ui.Center).Shrink(0)
	smoothFlat(track, col, 8, col.chip)
	track.Children(func() {
		for _, option := range []struct {
			key   string
			label string
		}{
			{"history", "下载历史"},
			{"files", "已打开"},
		} {
			option := option
			active := s.source == option.key
			seg := ui.ButtonBase(c).Width(78).Height(26).Radius(6).Center().
				Cursor(ui.CursorPointer).Label(option.label)
			if active {
				seg.Background(col.segThumb)
				if col.dark {
					seg.Shadow(0, 1, 1, 0, ui.RGBA(0, 0, 0, 0.25))
				} else {
					seg.Shadow(0, 0.5, 1.5, 0, ui.RGBA(0, 0, 0, 0.16)).
						Shadow(0, 0, 0, 0.5, ui.RGBA(0, 0, 0, 0.04))
				}
			}
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
		if entry.albumMID != "" {
			a.cover(c, col, entry.albumMID, 34)
		} else {
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
		if s.loading {
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
	title, hint := "下载历史为空", "下载过的歌曲会出现在这里，也可以直接打开本地文件。"
	if s.source == "files" {
		title, hint = "还没有打开文件", "打开本地的 MP3 / FLAC 文件后即可编辑标签。"
	} else if a.history.err != "" {
		title, hint = "下载历史不可用", a.history.err
	}
	ui.Column(c).Fill().Center().Children(func() {
		ui.Column(c).Width(420).Children(func() {
			emptyState(c, col, iconTag, title, hint)
			ui.Row(c).Gap(8).Justify(ui.Center).Children(func() {
				secondaryButton(c, col, iconFolder, "打开文件…", a.pickTagFiles)
				if s.source == "history" {
					secondaryButton(c, col, iconRefresh, "重新加载", a.retryTagHistory)
				}
			})
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

func (a *app) tagStudioForm(c *ui.Context, col colors, entry *tagEntry) {
	s := &a.tagStudio
	scroll := ui.Scroll(c).Key("tag-studio-editor").Grow(1).MinHeight(0).Gap(18)
	scroll.Children(func() {
		a.tagStudioSummary(c, col, entry)
		a.settingsGroup(c, col, "基本信息", func() {
			a.settingRow(c, col, "标题", "", true, func() {
				a.tagField(c, col, &s.draft.Title, "歌曲标题", 320)
			})
			a.settingRow(c, col, "艺人", "", false, func() {
				a.tagField(c, col, &s.draft.Artist, "歌手", 320)
			})
			a.settingRow(c, col, "专辑", "", false, func() {
				a.tagField(c, col, &s.draft.Album, "专辑名", 320)
			})
			divider(c, col)
			ui.Row(c).FillWidth().Gap(12).Padding(11, 0).AlignItems(ui.End).Children(func() {
				a.tagShortField(c, col, "音轨号", &s.trackText, "如 3 或 3/12")
				a.tagShortField(c, col, "碟号", &s.discText, "如 1 或 1/2")
				a.tagShortField(c, col, "日期", &s.draft.Date, "如 2024-01-01")
				a.tagShortField(c, col, "流派", &s.draft.Genre, "如 流行")
				a.tagShortField(c, col, "语言", &s.draft.Language, "如 国语")
			})
		})
		a.settingsGroup(c, col, "创作者", func() {
			ui.Row(c).FillWidth().Gap(12).Padding(11, 4).AlignItems(ui.End).Children(func() {
				a.tagShortField(c, col, "作曲", &s.draft.Composer, "作曲")
				a.tagShortField(c, col, "作词", &s.draft.Lyricist, "作词")
			})
		})
		a.settingsGroup(c, col, "封面", func() {
			a.tagStudioCover(c, col, entry)
		})
		a.settingsGroup(c, col, "歌词", func() {
			a.tagStudioLyrics(c, col, entry)
		})
	})
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

func (a *app) tagStudioSummary(c *ui.Context, col colors, entry *tagEntry) {
	s := &a.tagStudio
	softCard(c, col, func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconTag).Size(16, 16).TextColor(col.muted)
			ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, entry.displayTitle()).FontSize(14).FontWeight(600).
						TextColor(col.text).SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
					if s.dirty {
						badge := ui.Row(c).Height(18).Padding(0, 7).Radius(9).
							AlignItems(ui.Center).Shrink(0)
						badge.Background(col.accent.Alpha(0.12)).
							Border(1, col.accent.Alpha(0.30))
						badge.Children(func() {
							ui.Text(c, "未保存").FontSize(10).FontWeight(600).
								TextColor(col.accent).SingleLine()
						})
					}
				})
				ui.Text(c, entry.path).FontSize(11).TextColor(col.muted).
					SingleLine().Ellipsis("…").Tooltip(entry.path)
				detail := entry.formatLabel()
				if size := format.ByteSize(entry.size); size != "" {
					detail += " · " + size
				}
				ui.Text(c, detail).FontSize(11).TextColor(col.faint)
			})
			iconButton(c, col, iconChevronLeft, "上一个 (Alt+←)", func() { a.tagStudioStep(-1) })
			iconButton(c, col, iconChevronRight, "下一个 (Alt+→)", func() { a.tagStudioStep(1) })
			iconButton(c, col, iconFolder, "在文件夹中显示", func() {
				a.showHistoryFile(entry.path)
			})
			switch {
			case s.metaBusy:
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Spinner(c).Size(13, 13)
					ui.Text(c, "补全中…").FontSize(12).TextColor(col.muted)
				})
			case entry.songMID != "":
				secondaryButton(c, col, iconSparkles, "从 QQ 音乐补全", a.fillTagStudioMeta)
			default:
				secondaryButton(c, col, iconSearch, "搜索匹配", a.openTagMatch)
			}
		})
	})
}

func (a *app) tagStudioCover(c *ui.Context, col colors, entry *tagEntry) {
	s := &a.tagStudio
	ui.Row(c).FillWidth().Padding(11, 8).Gap(14).AlignItems(ui.Start).Children(func() {
		box := ui.Box(c).Size(128, 128).Radius(8).Background(col.cardAlt).
			Border(1, col.border).Clip()
		box.Children(func() {
			if s.cover != nil {
				ui.Image(c, s.cover).Size(128, 128).Fit(ui.Cover)
				return
			}
			ui.Row(c).Fill().Center().Children(func() {
				ui.Icon(c, iconDisc).Size(30, 30).TextColor(col.faint)
			})
		})
		ui.Column(c).Grow(1).MinWidth(0).Gap(8).Children(func() {
			ui.Text(c, "封面会写入 MP3 的 APIC 或 FLAC 的 PICTURE，仅支持 PNG / JPEG。").
				FontSize(12).TextColor(col.muted).LineHeight(1.4)
			if len(s.draft.Cover) > 0 {
				detail := format.ByteSize(int64(len(s.draft.Cover)))
				if s.draft.CoverMIME != "" {
					detail += " · " + s.draft.CoverMIME
				}
				ui.Text(c, detail).FontSize(11).TextColor(col.faint)
			} else {
				ui.Text(c, "当前没有封面").FontSize(11).TextColor(col.faint)
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ghostButton(c, col, iconFolder, "选择图片…", a.pickTagStudioCover)
				if s.coverBusy {
					ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
						ui.Spinner(c).Size(12, 12)
						ui.Text(c, "获取中…").FontSize(12).TextColor(col.muted)
					})
				} else {
					ghostButton(c, col, iconSparkles, "从 QQ 音乐获取", func() {
						if entry.albumMID == "" {
							a.setToast("该文件没有关联专辑，无法获取封面")
							return
						}
						a.fetchTagStudioCover()
					})
				}
				if len(s.draft.Cover) > 0 {
					ghostDangerButton(c, col, iconTrash, "移除", a.removeTagStudioCover)
				}
			})
		})
	})
}

func (a *app) tagStudioLyrics(c *ui.Context, col colors, entry *tagEntry) {
	s := &a.tagStudio
	ui.Column(c).FillWidth().Padding(11, 8).Gap(8).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
				ui.Text(c, "歌词文本").FontSize(13).TextColor(col.text)
				ui.Text(c, "LRC 文本会写入 USLT / LYRICS，留空保存会移除歌词。").
					FontSize(12).TextColor(col.muted)
			})
			if s.lyricBusy {
				ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
					ui.Spinner(c).Size(12, 12)
					ui.Text(c, "获取中…").FontSize(12).TextColor(col.muted)
				})
			} else {
				secondaryButton(c, col, iconSparkles, "获取歌词", a.requestTagLyrics)
			}
		})
		frame := ui.Column(c).FillWidth().Padding(8, 6)
		var input ui.Element
		frame.Children(func() {
			input = ui.TextAreaBase(c, &s.draft.Lyrics).FontSize(12).
				Lines(6, 12).FocusRing(false)
			if input.Changed() {
				s.dirty = true
			}
		})
		glow := frame.Animate("focus", b2f(input.Focused()), 120*time.Millisecond)
		smoothField(frame, col, 7, col.control, col.fieldStroke, col.accent, glow)
	})
}

func (a *app) tagField(c *ui.Context, col colors, value *string, placeholder string, width float32) {
	a.textField(c, col, value, placeholder, width, 30, func() {
		a.tagStudio.dirty = true
	}, nil)
}

// tagShortField is a label-above field for the compact rows of the tag
// form; it shares its row with its siblings.
func (a *app) tagShortField(c *ui.Context, col colors, label string, value *string, placeholder string) {
	ui.Column(c).Grow(1).MinWidth(110).Gap(4).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(col.muted)
		a.textField(c, col, value, placeholder, 0, 30, func() {
			a.tagStudio.dirty = true
		}, nil)
	})
}

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
	s.draft = tag.Info{}
	s.original = tag.Info{}
	s.cover = nil
	s.trackText = ""
	s.discText = ""
	s.loadedPath = ""
	s.dirty = false
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
	s.dirty = false
}

func (a *app) pickTagFiles() {
	win := a.win
	if win == nil {
		return
	}
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:   win,
			Title:    "打开音乐文件",
			Filters:  []mygo.FileFilter{{Name: "音乐文件", Extensions: []string{"mp3", "flac"}}},
			Multiple: true,
		})
		if err != nil {
			a.update(func() { a.toast = "打开文件选择器失败：" + err.Error() })
			return
		}
		if len(paths) == 0 {
			return
		}
		a.update(func() { a.addTagFiles(paths) })
	}()
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
		a.setToast("所选文件已在列表中，或不是 MP3 / FLAC")
		return
	}
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

func (a *app) fetchTagStudioCover() {
	s := &a.tagStudio
	entry := s.activeEntry(s.selected)
	if entry == nil || s.coverBusy {
		return
	}
	if entry.albumMID == "" {
		a.setToast("该文件没有关联专辑，无法获取封面")
		return
	}
	albumMID := entry.albumMID
	target := entry.path
	size := settings.CoverSizeValue(a.coverSize)
	s.coverBusy = true
	a.refresh()
	var data []byte
	var mime string
	var bitmap *ui.Bitmap
	a.async(&s.coverSlot, 30*time.Second, func(ctx context.Context) error {
		var err error
		data, mime, err = a.metadataCover(ctx, albumMID, size)
		if err != nil {
			return err
		}
		bitmap, err = ui.DecodeBitmap(data)
		return err
	}, func(err error) {
		s.coverBusy = false
		current := s.activeEntry(s.selected)
		if current == nil || !samePath(current.path, target) {
			a.refresh()
			return
		}
		if err != nil {
			a.toast = "获取封面失败：" + friendlyError(err)
		} else {
			s.draft.Cover = data
			s.draft.CoverMIME = mime
			s.cover = bitmap
			s.dirty = true
			a.toast = "封面已更新"
		}
		a.refresh()
	})
}

// openTagMatch starts a song search for the selected entry, so files the
// download history does not know can still use the QQ Music lookups.
func (a *app) openTagMatch() {
	s := &a.tagStudio
	entry := s.activeEntry(s.selected)
	if entry == nil {
		return
	}
	s.matchFor = entry.path
	s.matchQuery = ""
	s.matchResults = nil
	s.matchErr = ""
	s.matchOpen = true
	a.refresh()
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
// which is enough for the completion, lyric and cover lookups to work.
func (a *app) bindTagMatch(song qqmusic.Song) {
	s := &a.tagStudio
	entry := s.activeEntry(s.selected)
	if entry == nil || !samePath(entry.path, s.matchFor) {
		s.matchOpen = false
		a.refresh()
		return
	}
	entry.songMID = song.Mid
	entry.albumMID = song.AlbumMid
	s.matchOpen = false
	a.setToast("已关联「" + song.Title + "」，可继续补全信息")
}

func (a *app) tagMatchModal(c *ui.Context, col colors) {
	s := &a.tagStudio
	a.modal(c, col, &s.matchOpen, "搜索匹配歌曲", 460, func(close func()) {
		ui.Column(c).FillWidth().Gap(12).Children(func() {
			ui.Text(c, "选择 QQ 音乐中的歌曲关联到当前文件，之后即可补全信息、获取歌词和封面。").
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

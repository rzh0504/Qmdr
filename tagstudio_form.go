package main

// Tag Studio 编辑表单：顶部信息卡（封面 + 标题/歌手 + 操作）、基本信息分组、
// 歌词区。

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

func (a *app) tagStudioForm(c *ui.Context, col colors, entry *tagEntry) {
	s := &a.tagStudio
	// 两段式：摘要与基本信息固定在上，歌词区吃掉剩余高度——全屏时文本框
	// 拉伸填满而不是在下方留白。
	ui.Column(c).Fill().Gap(18).Children(func() {
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
			divider(c, col)
			ui.Row(c).FillWidth().Gap(12).Padding(11, 0).AlignItems(ui.End).Children(func() {
				a.tagShortField(c, col, "作曲", &s.draft.Composer, "作曲")
				a.tagShortField(c, col, "作词", &s.draft.Lyricist, "作词")
			})
		})
		lyrics := ui.Column(c).FillWidth().Grow(1).MinHeight(150).Gap(8)
		lyrics.Children(func() {
			groupTitle(c, col, "歌词")
			card := ui.Column(c).FillWidth().FillHeight().MinHeight(0).Padding(2, 16)
			smoothFlat(card, col, 12, col.chip)
			card.Children(func() {
				a.tagStudioLyrics(c, col, entry)
			})
		})
	})
}

func (a *app) tagStudioSummary(c *ui.Context, col colors, entry *tagEntry) {
	s := &a.tagStudio
	softCard(c, col, func() {
		// 固定行高：不同文件的字形走不同字体回退，行高会有±1px差异，
		// 钉死高度让卡片在切换时完全不动。
		ui.Row(c).Height(50).Gap(12).AlignItems(ui.Center).Children(func() {
			// The cover: left-click picks a new image, right-click opens the
			// menu that removes the current one.
			cover := ui.ButtonBase(c).Size(34, 34).Radius(4).Background(col.cardAlt).
				Border(1, col.border).Clip().Cursor(ui.CursorPointer).
				Label("更换封面").Tooltip("点击更换封面，右键移除（PNG / JPEG，8 MB 内）")
			if cover.Hovered() {
				cover.Border(1, col.accent)
			}
			cover.Children(func() {
				if s.cover != nil {
					ui.Image(c, s.cover).Size(34, 34).Fit(ui.Cover)
				} else {
					ui.Row(c).Fill().Center().Children(func() {
						ui.Icon(c, iconDisc).Size(15, 15).TextColor(col.faint)
					})
				}
			})
			if cover.Clicked() {
				a.pickTagStudioCover()
			}
			if cover.RightClicked() && len(s.draft.Cover) > 0 {
				b := cover.Bounds()
				x, y, over := cover.PointerPosition()
				if !over {
					x, y = b.W/2, b.H/2
				}
				a.contextMenu = contextMenuState{open: true, x: b.X + x, y: b.Y + y, items: []menuAction{
					{label: "移除封面", run: a.removeTagStudioCover},
				}}
				a.refresh()
			}
			ui.Column(c).Grow(1).MinWidth(0).FillHeight().Gap(2).Children(func() {
				ui.Row(c).FillWidth().Height(22).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, a.tagSummaryTitle(entry)).FontSize(14).FontWeight(600).
						TextColor(col.text).SingleLine().Ellipsis("…").
						MaxWidthPercent(60).Shrink(0)
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
					ui.Spacer(c)
				})
				ui.Row(c).FillWidth().Height(18).AlignItems(ui.Center).Children(func() {
					ui.Text(c, a.tagSummaryDetail(entry)).FontSize(12).TextColor(col.muted).
						SingleLine().Ellipsis("…")
				})
			})
			ui.Row(c).Gap(8).Shrink(0).AlignItems(ui.Center).Children(func() {
				switch {
				case s.metaBusy:
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Spinner(c).Size(13, 13)
						ui.Text(c, "补全中…").FontSize(12).TextColor(col.muted)
					})
				case entry.songMID != "":
					secondaryButton(c, col, nil, "从 QQ 音乐补全", a.fillTagStudioMeta)
				default:
					secondaryButton(c, col, iconSearch, "搜索匹配", a.openTagMatch)
				}
				if len(s.draft.Cover) > 0 {
					iconButton(c, col, iconTrash, "移除封面", a.removeTagStudioCover)
				}
				switch {
				case s.saving:
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Spinner(c).Size(13, 13)
						ui.Text(c, "保存中…").FontSize(12).TextColor(col.muted)
					})
				case s.dirty:
					secondaryButton(c, col, iconRefresh, "还原", a.revertTagStudio)
					primaryButton(c, col, iconCheck, "保存", a.saveTagStudio)
				default:
					primaryButtonDisabled(c, col, iconCheck, "保存")
				}
			})
		})
	})
}

// tagSummaryTitle prefers the draft, so the header updates as the user types.
func (a *app) tagSummaryTitle(entry *tagEntry) string {
	if title := strings.TrimSpace(a.tagStudio.draft.Title); title != "" {
		return title
	}
	return entry.displayTitle()
}

// tagSummaryDetail is the artist · album line, also draft-first.
func (a *app) tagSummaryDetail(entry *tagEntry) string {
	s := &a.tagStudio
	artist := strings.TrimSpace(s.draft.Artist)
	if artist == "" {
		artist = entry.displayArtist()
	}
	album := strings.TrimSpace(s.draft.Album)
	if album == "" {
		album = entry.displayAlbum()
	}
	switch {
	case artist == "":
		return album
	case album == "":
		return artist
	}
	return artist + " · " + album
}

func (a *app) tagStudioLyrics(c *ui.Context, col colors, entry *tagEntry) {
	s := &a.tagStudio
	ui.Column(c).FillWidth().FillHeight().Padding(11, 8).Gap(8).Children(func() {
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
				secondaryButton(c, col, nil, "获取歌词", a.requestTagLyrics)
			}
		})
		frame := ui.Column(c).FillWidth().FillHeight().MinHeight(96).Padding(8, 6)
		var input ui.Element
		frame.Children(func() {
			input = ui.TextAreaBase(c, &s.draft.Lyrics).FontSize(12).
				Lines(6, 12).FocusRing(false).FillHeight()
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

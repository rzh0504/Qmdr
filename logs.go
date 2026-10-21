package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/rzh0504/qmdr/internal/history"
)

const (
	logExportDays    = 7
	logRetentionDays = 30
)

func (a *app) storeLog(entry LogEntry) {
	store := a.history.store
	if store == nil {
		return
	}
	a.dispatchDisk(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := store.AddLog(ctx, history.LogRecord{
			At:      entry.At,
			Level:   entry.Level,
			Message: entry.Message,
		}); err != nil {
			log.Printf("write task log: %v", err)
		}
	})
}

func (a *app) flushLogBacklog() {
	if a.history.store == nil || len(a.history.logBacklog) == 0 {
		return
	}
	entries := a.history.logBacklog
	a.history.logBacklog = nil
	for _, entry := range entries {
		a.storeLog(entry)
	}
}

func (a *app) exportLogs() {
	store := a.history.store
	if store == nil {
		a.setToast("日志数据库尚未就绪，请稍后再试")
		return
	}
	win := a.win
	if win == nil {
		return
	}
	go func() {
		since := time.Now().AddDate(0, 0, -logExportDays)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		records, err := store.LogsSince(ctx, since)
		if err != nil {
			a.update(func() { a.toast = "读取任务日志失败：" + friendlyError(err) })
			return
		}
		if len(records) == 0 {
			a.update(func() {
				a.toast = fmt.Sprintf("最近 %d 天没有任务日志", logExportDays)
			})
			return
		}
		selected, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent:      win,
			Title:       "导出任务日志",
			DefaultPath: "Qmdr-logs-" + time.Now().Format("20060102-1504") + ".txt",
			Filters:     []mygo.FileFilter{{Name: "文本文件", Extensions: []string{"txt"}}},
		})
		if err != nil {
			a.update(func() { a.toast = "导出任务日志失败：" + err.Error() })
			return
		}
		if selected == "" {
			return
		}
		if !strings.HasSuffix(strings.ToLower(selected), ".txt") {
			selected += ".txt"
		}
		exportedAt := time.Now()
		content := buildLogText(records, since, exportedAt)
		if err := os.WriteFile(selected, []byte(content), 0o600); err != nil {
			a.update(func() { a.toast = "写入任务日志失败：" + friendlyError(err) })
			return
		}
		a.update(func() {
			a.toast = fmt.Sprintf("已导出 %d 条任务日志", len(records))
		})
	}()
}

func buildLogText(records []history.LogRecord, since, exportedAt time.Time) string {
	var b strings.Builder
	b.WriteString("Qmdr 任务日志\n")
	b.WriteString("导出时间：" + exportedAt.Format("2006-01-02 15:04:05") + "\n")
	b.WriteString("时间范围：" + since.Format("2006-01-02 15:04:05") +
		" ~ " + exportedAt.Format("2006-01-02 15:04:05") + "\n")
	b.WriteString(strings.Repeat("-", 48) + "\n")
	for _, record := range records {
		b.WriteString(record.At.Local().Format("2006-01-02 15:04:05"))
		b.WriteString(" [")
		b.WriteString(logLevelLabel(record.Level))
		b.WriteString("] ")
		b.WriteString(record.Message)
		b.WriteByte('\n')
	}
	return b.String()
}

func logLevelLabel(level string) string {
	switch level {
	case "error":
		return "错误"
	case "warning":
		return "警告"
	case "success":
		return "成功"
	default:
		return "信息"
	}
}

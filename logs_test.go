package main

import (
	"strings"
	"testing"
	"time"

	"github.com/rzh0504/qmdr/internal/history"
)

func TestBuildLogText(t *testing.T) {
	exportedAt := time.Unix(1700003600, 0)
	text := buildLogText([]history.LogRecord{
		{At: time.Unix(1700000000, 0), Level: "warning", Message: "disk almost full"},
	}, exportedAt.AddDate(0, 0, -logExportDays), exportedAt)

	if !strings.Contains(text, "Qmdr 任务日志") {
		t.Fatalf("log header missing: %q", text)
	}
	if !strings.Contains(text, "时间范围：") {
		t.Fatalf("log range missing: %q", text)
	}
	if !strings.Contains(text, "[警告] disk almost full") {
		t.Fatalf("log line missing: %q", text)
	}
}

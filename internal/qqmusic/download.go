package qqmusic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ProgressFunc func(received, total int64)

const downloadStallTimeout = 90 * time.Second

func (c *Client) Download(ctx context.Context, rawURL, dest string, overwrite bool, progress ProgressFunc) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return 0, err
	}
	downloadCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stall := time.AfterFunc(downloadStallTimeout, cancel)
	defer stall.Stop()

	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", defaultUA)
	req.Header.Set("Referer", "https://y.qq.com/")

	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		if downloadCtx.Err() != nil {
			return 0, fmt.Errorf("下载超时：长时间没有收到数据")
		}
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("下载服务器返回 HTTP %d", resp.StatusCode)
	}

	tempPath := dest + ".part"
	file, err := os.Create(tempPath)
	if err != nil {
		return 0, err
	}
	success := false
	defer func() {
		_ = file.Close()
		if !success {
			_ = os.Remove(tempPath)
		}
	}()

	total := resp.ContentLength
	var received int64
	buf := make([]byte, 256<<10)
	lastReport := time.Now()
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			stall.Reset(downloadStallTimeout)
			if _, err := file.Write(buf[:n]); err != nil {
				return received, err
			}
			received += int64(n)
			if progress != nil && (time.Since(lastReport) >= 120*time.Millisecond) {
				progress(received, total)
				lastReport = time.Now()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			if ctx.Err() != nil {
				return received, ctx.Err()
			}
			if downloadCtx.Err() != nil {
				return received, fmt.Errorf("下载超时：长时间没有收到数据")
			}
			return received, readErr
		}
		if err := ctx.Err(); err != nil {
			return received, err
		}
	}

	if err := ctx.Err(); err != nil {
		return received, err
	}
	if received == 0 {
		return 0, fmt.Errorf("下载内容为空")
	}
	if total > 0 && received != total {
		return received, fmt.Errorf("下载不完整：已接收 %d/%d 字节", received, total)
	}
	if err := file.Sync(); err != nil {
		return received, err
	}
	if err := file.Close(); err != nil {
		return received, err
	}
	if _, err := os.Stat(dest); err == nil && !overwrite {
		// The file appeared while this one was downloading. Keep the
		// completed audio under the next free name instead of failing
		// after a full download.
		dup := ""
		ext := filepath.Ext(dest)
		stem := strings.TrimSuffix(dest, ext)
		for i := 2; i <= 999; i++ {
			candidate := fmt.Sprintf("%s (%d)%s", stem, i, ext)
			if _, err := os.Stat(candidate); err != nil {
				dup = candidate
				break
			}
		}
		if dup == "" {
			return received, fmt.Errorf("目标文件已存在：%s", filepath.Base(dest))
		}
		if err := os.Rename(tempPath, dup); err != nil {
			return received, err
		}
		success = true
		return received, nil
	}
	if err := os.Rename(tempPath, dest); err != nil {
		return received, err
	}
	success = true
	if progress != nil {
		progress(received, total)
	}
	return received, nil
}

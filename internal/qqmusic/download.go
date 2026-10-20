package qqmusic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type ProgressFunc func(received, total int64)

func (c *Client) Download(ctx context.Context, rawURL, dest string, overwrite bool, progress ProgressFunc) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", defaultUA)
	req.Header.Set("Referer", "https://y.qq.com/")

	resp, err := c.http.Do(req)
	if err != nil {
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
			return received, readErr
		}
		if err := ctx.Err(); err != nil {
			return received, err
		}
	}

	if err := file.Sync(); err != nil {
		return received, err
	}
	if err := file.Close(); err != nil {
		return received, err
	}
	if overwrite {
		_ = os.Remove(dest)
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

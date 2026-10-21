package qqmusic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

var ErrNoCover = errors.New("没有封面")

var albumMidPattern = regexp.MustCompile(`^[A-Za-z0-9]+$`)

func (c *Client) Cover(ctx context.Context, albumMid string, size int) ([]byte, string, error) {
	if albumMid == "" || !albumMidPattern.MatchString(albumMid) {
		return nil, "", ErrNoCover
	}
	switch size {
	case 150, 300, 500, 800:
	default:
		size = 800
	}
	rawURL := fmt.Sprintf("https://y.gtimg.cn/music/photo_new/T002R%dx%dM000%s.jpg", size, size, albumMid)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", defaultUA)
	req.Header.Set("Referer", "https://y.qq.com/")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("封面服务器返回 HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, "", err
	}
	if len(data) < 1024 {
		return nil, "", ErrNoCover
	}
	switch {
	case len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n":
		return data, "image/png", nil
	case len(data) >= 2 && data[0] == 0xff && data[1] == 0xd8:
		return data, "image/jpeg", nil
	default:
		return nil, "", ErrNoCover
	}
}

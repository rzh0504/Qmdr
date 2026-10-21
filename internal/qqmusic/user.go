package qqmusic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// UserProfile is the signed-in user's public profile.
type UserProfile struct {
	Nick      string
	AvatarURL string
}

// UserProfile reads the signed-in user's nickname and avatar. The avatar
// is a URL to fetch separately.
func (c *Client) UserProfile(ctx context.Context) (UserProfile, error) {
	if !c.HasCredential() {
		return UserProfile{}, errors.New("需要登录后才能获取用户资料")
	}
	var data struct {
		Info struct {
			Nick string `json:"nick"`
			Logo string `json:"logo"`
		} `json:"info"`
	}
	if err := c.call(ctx, "music.UserInfo.userInfoServer", "GetLoginUserInfo", map[string]any{}, &data); err != nil {
		return UserProfile{}, err
	}
	return UserProfile{
		Nick:      cleanText(data.Info.Nick),
		AvatarURL: strings.TrimSpace(data.Info.Logo),
	}, nil
}

// FetchImage downloads an image the API points to, such as an avatar.
func (c *Client) FetchImage(ctx context.Context, rawURL string) ([]byte, error) {
	rawURL = strings.TrimSpace(rawURL)
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" || parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("图片地址无效")
	}
	host := strings.ToLower(parsed.Hostname())
	if !strings.HasSuffix(host, ".qlogo.cn") && !strings.HasSuffix(host, ".qq.com") {
		return nil, errors.New("图片地址不在允许的域名内")
	}
	// QQ's avatar host serves the same image over HTTPS.
	if parsed.Scheme == "http" {
		parsed.Scheme = "https"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", defaultUA)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("图片服务器返回 HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("图片内容为空")
	}
	return data, nil
}

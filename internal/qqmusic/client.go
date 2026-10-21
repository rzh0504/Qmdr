package qqmusic

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/net/publicsuffix"
)

const (
	apiEndpoint     = "https://u.y.qq.com/cgi-bin/musicu.fcg"
	cdnBase         = "https://isure.stream.qqmusic.qq.com/"
	defaultUA       = "Mozilla/5.0 (Windows NT 11.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 Edg/116.0.1938.54"
	fallbackQIMEI   = "6c9d3cd110abca9b16311cee10001e717614"
	versionCode     = 13020508
	maxResponseSize = 32 << 20
)

type APIError struct {
	Module  string
	Method  string
	Code    int
	Message string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("qqmusic: %s.%s returned code %d: %s", e.Module, e.Method, e.Code, e.Message)
	}
	return fmt.Sprintf("qqmusic: %s.%s returned code %d", e.Module, e.Method, e.Code)
}

type Credential struct {
	OpenID       string `json:"openid"`
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	ExpiredAt    int64  `json:"expired_at"`
	MusicID      int64  `json:"musicid"`
	Musickey     string `json:"musickey"`
	UnionID      string `json:"unionid"`
	StrMusicID   string `json:"str_musicid"`
	RefreshKey   string `json:"refresh_key"`
	EncryptUin   string `json:"encrypt_uin"`
	LoginType    int    `json:"login_type"`
}

func (c Credential) Valid() bool {
	return c.MusicID != 0 && c.Musickey != ""
}

type Client struct {
	http *http.Client

	Credential Credential
	QIMEI      string
}

func New() *Client {
	c := &Client{
		http:  newHTTPClient(),
		QIMEI: fallbackQIMEI,
	}
	c.Credential = loadCredential()
	return c
}

func newHTTPClient() *http.Client {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 8
	return &http.Client{Transport: transport, Jar: jar}
}

func CredentialPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Qmdr", "qqmusic_cred.json")
}

func loadCredential() Credential {
	path := CredentialPath()
	if path == "" {
		return Credential{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Credential{}
	}
	var c Credential
	if err := json.Unmarshal(data, &c); err != nil || !c.Valid() {
		return Credential{}
	}
	return c
}

func (c *Client) SaveCredential() error {
	path := CredentialPath()
	if path == "" {
		return fmt.Errorf("找不到配置目录")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c.Credential, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (c *Client) ReloadCredential() bool {
	c.Credential = loadCredential()
	return c.Credential.Valid()
}

func (c *Client) HasCredential() bool {
	return c.Credential.Valid()
}

func (c *Client) UserID() string {
	if c.Credential.MusicID == 0 {
		return ""
	}
	return strconv.FormatInt(c.Credential.MusicID, 10)
}

func (c *Client) call(ctx context.Context, module, method string, params any, out any) error {
	return c.callWith(ctx, c.http, module, method, params, nil, out)
}

func (c *Client) callWith(ctx context.Context, hc *http.Client, module, method string, params any, extraCommon map[string]any, out any) error {
	comm := map[string]any{
		"cv":         versionCode,
		"v":          versionCode,
		"QIMEI36":    c.QIMEI,
		"ct":         "11",
		"tmeAppID":   "qqmusic",
		"format":     "json",
		"inCharset":  "utf-8",
		"outCharset": "utf-8",
		"uid":        "3931641530",
	}
	if c.HasCredential() {
		comm["qq"] = strconv.FormatInt(c.Credential.MusicID, 10)
		comm["authst"] = c.Credential.Musickey
		comm["tmeLoginType"] = strconv.Itoa(c.Credential.LoginType)
	}
	for k, v := range extraCommon {
		comm[k] = v
	}

	key := module + "." + method
	payload := map[string]any{
		"comm": comm,
		key: map[string]any{
			"module": module,
			"method": method,
			"param":  params,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiEndpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", defaultUA)
	req.Header.Set("Referer", "https://y.qq.com/")
	if c.HasCredential() {
		req.AddCookie(&http.Cookie{Name: "uin", Value: strconv.FormatInt(c.Credential.MusicID, 10)})
		req.AddCookie(&http.Cookie{Name: "qqmusic_key", Value: c.Credential.Musickey})
		req.AddCookie(&http.Cookie{Name: "qm_keyst", Value: c.Credential.Musickey})
		req.AddCookie(&http.Cookie{Name: "tmeLoginType", Value: strconv.Itoa(c.Credential.LoginType)})
	}

	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qqmusic: HTTP %d from %s", resp.StatusCode, apiEndpoint)
	}
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return err
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return fmt.Errorf("qqmusic: decode response: %w", err)
	}
	raw, ok := envelope[key]
	if !ok {
		return fmt.Errorf("qqmusic: response has no %s", key)
	}
	var result struct {
		Code    int             `json:"code"`
		Message string          `json:"msg"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return fmt.Errorf("qqmusic: decode %s: %w", key, err)
	}
	if result.Code != 0 {
		return &APIError{Module: module, Method: method, Code: result.Code, Message: result.Message}
	}
	if out != nil && len(result.Data) > 0 {
		if err := json.Unmarshal(result.Data, out); err != nil {
			return fmt.Errorf("qqmusic: decode %s data: %w", key, err)
		}
	}
	return nil
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(buf)
}

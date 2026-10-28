package qqmusic

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/publicsuffix"

	"github.com/rzh0504/qmdr/internal/fsutil"
)

const (
	cdnBase         = "https://isure.stream.qqmusic.qq.com/"
	defaultUA       = "Mozilla/5.0 (Windows NT 11.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 Edg/116.0.1938.54"
	fallbackQIMEI   = "6c9d3cd110abca9b16311cee10001e717614"
	placeholderUID  = "0"
	versionCode     = 13020508
	maxResponseSize = 32 << 20

	// Refresh slightly early so a request cannot race the expiry timestamp.
	credentialExpiryMargin = 10 * time.Minute
)

var apiEndpoint = "https://u.y.qq.com/cgi-bin/musicu.fcg"

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

	MusickeyCreateTime int64 `json:"musickeyCreateTime"`
	KeyExpiresIn       int64 `json:"keyExpiresIn"`
}

func (c Credential) Valid() bool {
	return c.MusicID != 0 && c.Musickey != ""
}

// Expired reports whether the refresh window recorded in expired_at has
// closed. expired_at is normally in seconds; millisecond values are accepted.
func (c Credential) Expired(now time.Time) bool {
	if c.ExpiredAt <= 0 {
		return false
	}
	expires := c.ExpiredAt
	if expires > 1e12 {
		expires /= 1000
	}
	if expires < 1e9 {
		return false
	}
	return !now.Add(credentialExpiryMargin).Before(time.Unix(expires, 0))
}

// NeedsRefresh reports whether the musickey is about to expire. The lifetime
// comes from keyExpiresIn, which is only a few days long.
func (c Credential) NeedsRefresh(now time.Time) bool {
	if c.MusickeyCreateTime <= 0 || c.KeyExpiresIn <= 0 {
		return false
	}
	expires := c.MusickeyCreateTime + c.KeyExpiresIn
	return !now.Add(credentialExpiryMargin).Before(time.Unix(expires, 0))
}

type Client struct {
	http *http.Client

	mu                  sync.RWMutex
	credential          Credential
	credentialSeq       uint64
	onCredentialExpired func()
	saveErr             error

	refreshMu sync.Mutex

	QIMEI string
}

func New() *Client {
	c := &Client{
		http:  newHTTPClient(),
		QIMEI: fallbackQIMEI,
	}
	c.credential = loadCredential()
	return c
}

func (c *Client) Credential() Credential {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.credential
}

func (c *Client) setCredential(credential Credential) {
	c.mu.Lock()
	c.credential = credential
	c.credentialSeq++
	c.mu.Unlock()
}

func (c *Client) credentialVersion() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.credentialSeq
}

func (c *Client) setCredentialSaveError(err error) {
	c.mu.Lock()
	c.saveErr = err
	c.mu.Unlock()
}

func (c *Client) SetCredentialExpiredHandler(fn func()) {
	c.mu.Lock()
	c.onCredentialExpired = fn
	c.mu.Unlock()
}

// expireCredential drops a credential the server rejected and lets the UI
// switch to the signed-out state. The saved file is removed so the next
// start does not retry a token that is already known to be dead.
func (c *Client) expireCredential() {
	c.mu.Lock()
	if !c.credential.Valid() {
		c.mu.Unlock()
		return
	}
	c.credential = Credential{}
	c.credentialSeq++
	handler := c.onCredentialExpired
	c.mu.Unlock()
	if path := CredentialPath(); path != "" {
		_ = os.Remove(path)
	}
	if handler != nil {
		handler()
	}
}

func (c *Client) CredentialSaveError() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.saveErr
}

func newHTTPClient() *http.Client {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 8
	transport.ResponseHeaderTimeout = 20 * time.Second
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
	data, err := json.MarshalIndent(c.Credential(), "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, data, 0o600)
}

func (c *Client) ReloadCredential() bool {
	credential := loadCredential()
	c.setCredential(credential)
	return c.HasCredential()
}

func (c *Client) HasCredential() bool {
	_, ok := c.usableCredential()
	return ok
}

// usableCredential hides credentials that are missing or already expired
// locally, so callers never send a token past its recorded lifetime.
func (c *Client) usableCredential() (Credential, bool) {
	credential := c.Credential()
	if !credential.Valid() || credential.Expired(time.Now()) {
		return Credential{}, false
	}
	return credential, true
}

func (c *Client) UserID() string {
	credential, ok := c.usableCredential()
	if !ok {
		return ""
	}
	return strconv.FormatInt(credential.MusicID, 10)
}

func (c *Client) call(ctx context.Context, module, method string, params any, out any) error {
	return c.callWith(ctx, c.http, module, method, params, nil, out)
}

func (c *Client) callWith(ctx context.Context, hc *http.Client, module, method string, params any, extraCommon map[string]any, out any) error {
	if stored := c.Credential(); stored.Valid() && stored.Expired(time.Now()) {
		c.expireCredential()
	}
	version := c.credentialVersion()
	err := c.callOnce(ctx, hc, module, method, params, extraCommon, out)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != 1000 {
		return err
	}
	refreshed, refreshErr := c.refreshCredential(ctx, version)
	if refreshed {
		err = c.callOnce(ctx, hc, module, method, params, extraCommon, out)
		var retryErr *APIError
		if errors.As(err, &retryErr) && retryErr.Code == 1000 {
			c.expireCredential()
		}
		return err
	}
	if refreshErr == nil {
		c.expireCredential()
		return err
	}
	var refreshAPIErr *APIError
	if errors.As(refreshErr, &refreshAPIErr) && refreshAPIErr.Code == 1000 {
		c.expireCredential()
		return err
	}
	return refreshErr
}

func (c *Client) callOnce(ctx context.Context, hc *http.Client, module, method string, params any, extraCommon map[string]any, out any) error {
	comm := map[string]any{
		"cv":         versionCode,
		"v":          versionCode,
		"QIMEI36":    c.QIMEI,
		"ct":         "11",
		"tmeAppID":   "qqmusic",
		"format":     "json",
		"inCharset":  "utf-8",
		"outCharset": "utf-8",
		"uid":        placeholderUID,
	}
	credential, hasCredential := c.usableCredential()
	if hasCredential {
		comm["qq"] = strconv.FormatInt(credential.MusicID, 10)
		comm["authst"] = credential.Musickey
		comm["tmeLoginType"] = strconv.Itoa(credential.LoginType)
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
	if hasCredential {
		req.AddCookie(&http.Cookie{Name: "uin", Value: strconv.FormatInt(credential.MusicID, 10)})
		req.AddCookie(&http.Cookie{Name: "qqmusic_key", Value: credential.Musickey})
		req.AddCookie(&http.Cookie{Name: "qm_keyst", Value: credential.Musickey})
		req.AddCookie(&http.Cookie{Name: "tmeLoginType", Value: strconv.Itoa(credential.LoginType)})
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

// EnsureCredential refreshes a musickey that is about to expire before the
// app starts using it. It makes no request when the credential is fresh.
func (c *Client) EnsureCredential(ctx context.Context) error {
	credential, ok := c.usableCredential()
	if !ok || !credential.NeedsRefresh(time.Now()) {
		return nil
	}
	refreshed, err := c.refreshCredential(ctx, c.credentialVersion())
	if refreshed {
		return nil
	}
	if err == nil {
		c.expireCredential()
		return errors.New("登录已过期，请重新登录")
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.Code == 1000 {
		c.expireCredential()
	}
	return err
}

// refreshCredential trades refresh_key and refresh_token for a new musickey.
// expectedVersion prevents concurrent 1000 responses from refreshing twice.
func (c *Client) refreshCredential(ctx context.Context, expectedVersion uint64) (bool, error) {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()
	if c.credentialVersion() != expectedVersion {
		return c.HasCredential(), nil
	}
	old := c.Credential()
	if !old.Valid() || old.Expired(time.Now()) {
		return false, nil
	}
	if old.RefreshKey == "" || old.RefreshToken == "" {
		return false, nil
	}
	var payload credentialPayload
	params := map[string]any{
		"refresh_key":   old.RefreshKey,
		"refresh_token": old.RefreshToken,
		"musickey":      old.Musickey,
		"musicid":       old.MusicID,
	}
	extra := map[string]any{"tmeLoginType": strconv.Itoa(old.LoginType)}
	if err := c.callOnce(ctx, c.http, "music.login.LoginServer", "Login", params, extra, &payload); err != nil {
		return false, err
	}
	fresh := payload.credential(old.LoginType)
	if !fresh.Valid() {
		return false, errors.New("刷新登录状态响应缺少凭证字段")
	}
	c.setCredential(mergeCredential(old, fresh))
	c.setCredentialSaveError(c.SaveCredential())
	return true, nil
}

func mergeCredential(old, fresh Credential) Credential {
	if fresh.OpenID == "" {
		fresh.OpenID = old.OpenID
	}
	if fresh.RefreshToken == "" {
		fresh.RefreshToken = old.RefreshToken
	}
	if fresh.AccessToken == "" {
		fresh.AccessToken = old.AccessToken
	}
	if fresh.ExpiredAt == 0 {
		fresh.ExpiredAt = old.ExpiredAt
	}
	if fresh.MusicID == 0 {
		fresh.MusicID = old.MusicID
	}
	if fresh.Musickey == "" {
		fresh.Musickey = old.Musickey
	}
	if fresh.UnionID == "" {
		fresh.UnionID = old.UnionID
	}
	if fresh.StrMusicID == "" {
		fresh.StrMusicID = old.StrMusicID
	}
	if fresh.RefreshKey == "" {
		fresh.RefreshKey = old.RefreshKey
	}
	if fresh.EncryptUin == "" {
		fresh.EncryptUin = old.EncryptUin
	}
	if fresh.LoginType == 0 {
		fresh.LoginType = old.LoginType
	}
	if fresh.MusickeyCreateTime == 0 {
		if fresh.Musickey != old.Musickey {
			fresh.MusickeyCreateTime = time.Now().Unix()
		} else {
			fresh.MusickeyCreateTime = old.MusickeyCreateTime
		}
	}
	if fresh.KeyExpiresIn == 0 {
		fresh.KeyExpiresIn = old.KeyExpiresIn
	}
	return fresh
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(buf)
}

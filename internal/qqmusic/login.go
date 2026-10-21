package qqmusic

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type LoginEvent int

const (
	LoginWaiting LoginEvent = iota
	LoginScanned
	LoginConfirmed
	LoginExpired
	LoginRefused
	LoginDone
	LoginOther
)

type LoginResult struct {
	Event   LoginEvent
	Message string
}

type QRLogin struct {
	Kind  string
	Image []byte
	Mime  string

	identifier string
	client     *http.Client
}

func (c *Client) StartQRLogin(ctx context.Context, kind string) (*QRLogin, error) {
	kind = normalizeLoginKind(kind)
	hc := newHTTPClient()
	if kind == "wx" {
		return c.startWeChatQR(ctx, hc)
	}
	return c.startQQQR(ctx, hc)
}

func (c *Client) PollQRLogin(ctx context.Context, qr *QRLogin) (LoginResult, error) {
	if qr == nil {
		return LoginResult{Event: LoginOther, Message: "登录会话不存在"}, fmt.Errorf("登录会话不存在")
	}
	if normalizeLoginKind(qr.Kind) == "wx" {
		return c.pollWeChatQR(ctx, qr)
	}
	return c.pollQQQR(ctx, qr)
}

func normalizeLoginKind(kind string) string {
	k := strings.ToLower(strings.TrimSpace(kind))
	if strings.Contains(k, "wx") || strings.Contains(k, "微信") || strings.Contains(k, "wechat") {
		return "wx"
	}
	return "qq"
}

func (c *Client) startQQQR(ctx context.Context, hc *http.Client) (*QRLogin, error) {
	values := url.Values{
		"appid":      {"716027609"},
		"e":          {"2"},
		"l":          {"M"},
		"s":          {"3"},
		"d":          {"72"},
		"v":          {"4"},
		"t":          {strconv.FormatFloat(randFloat(), 'f', -1, 64)},
		"daid":       {"383"},
		"pt_3rd_aid": {"100497308"},
	}
	resp, err := get(ctx, hc, "https://ssl.ptlogin2.qq.com/ptqrshow?"+values.Encode(), map[string]string{
		"Referer": "https://xui.ptlogin2.qq.com/",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	qrsig := ""
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "qrsig" {
			qrsig = cookie.Value
			break
		}
	}
	if qrsig == "" {
		qrsig = cookieValue(hc, []string{"https://ssl.ptlogin2.qq.com/"}, "qrsig")
	}
	if qrsig == "" || len(body) == 0 {
		return nil, fmt.Errorf("获取 QQ 二维码失败")
	}
	return &QRLogin{Kind: "qq", Image: body, Mime: "image/png", identifier: qrsig, client: hc}, nil
}

func (c *Client) startWeChatQR(ctx context.Context, hc *http.Client) (*QRLogin, error) {
	values := url.Values{
		"appid":         {"wx48db31d50e334801"},
		"redirect_uri":  {"https://y.qq.com/portal/wx_redirect.html?login_type=2&surl=https://y.qq.com/"},
		"response_type": {"code"},
		"scope":         {"snsapi_login"},
		"state":         {"STATE"},
		"href":          {"https://y.qq.com/mediastyle/music_v17/src/css/popup_wechat.css#wechat_redirect"},
	}
	resp, err := get(ctx, hc, "https://open.weixin.qq.com/connect/qrconnect?"+values.Encode(), nil)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	match := regexp.MustCompile(`uuid=(.+?)"`).FindSubmatch(body)
	if len(match) < 2 {
		return nil, fmt.Errorf("获取微信登录标识失败")
	}
	identifier := strings.TrimSpace(string(match[1]))
	resp, err = get(ctx, hc, "https://open.weixin.qq.com/connect/qrcode/"+url.PathEscape(identifier), map[string]string{
		"Referer": "https://open.weixin.qq.com/connect/qrconnect",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	image, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if len(image) == 0 {
		return nil, fmt.Errorf("获取微信二维码失败")
	}
	return &QRLogin{Kind: "wx", Image: image, Mime: "image/jpeg", identifier: identifier, client: hc}, nil
}

func (c *Client) pollQQQR(ctx context.Context, qr *QRLogin) (LoginResult, error) {
	values := url.Values{
		"u1":         {"https://graph.qq.com/oauth2.0/login_jump"},
		"ptqrtoken":  {strconv.FormatUint(uint64(hash33(qr.identifier, 0)), 10)},
		"ptredirect": {"0"},
		"h":          {"1"},
		"t":          {"1"},
		"g":          {"1"},
		"from_ui":    {"1"},
		"ptlang":     {"2052"},
		"action":     {fmt.Sprintf("0-0-%d", time.Now().UnixMilli())},
		"js_ver":     {"20102616"},
		"js_type":    {"1"},
		"pt_uistyle": {"40"},
		"aid":        {"716027609"},
		"daid":       {"383"},
		"pt_3rd_aid": {"100497308"},
		"has_onekey": {"1"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ssl.ptlogin2.qq.com/ptqrlogin?"+values.Encode(), nil)
	if err != nil {
		return LoginResult{}, err
	}
	req.Header.Set("User-Agent", defaultUA)
	req.Header.Set("Referer", "https://xui.ptlogin2.qq.com/")
	req.AddCookie(&http.Cookie{Name: "qrsig", Value: qr.identifier})
	resp, err := qr.client.Do(req)
	if err != nil {
		return LoginResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return LoginResult{}, fmt.Errorf("登录服务器返回 HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return LoginResult{}, err
	}
	match := regexp.MustCompile(`ptuiCB\((.*?)\)`).FindSubmatch(body)
	if len(match) < 2 {
		return LoginResult{}, fmt.Errorf("获取二维码状态失败")
	}
	parts := strings.Split(string(match[1]), ",")
	for i := range parts {
		parts[i] = strings.Trim(strings.TrimSpace(parts[i]), "'")
	}
	if len(parts) == 0 {
		return LoginResult{Event: LoginOther, Message: "未知登录状态"}, nil
	}
	code, err := strconv.Atoi(parts[0])
	if err != nil {
		return LoginResult{Event: LoginOther, Message: "未知登录状态"}, nil
	}
	switch code {
	case 0, 405:
		if len(parts) < 3 {
			return LoginResult{}, fmt.Errorf("登录成功但响应缺少跳转地址")
		}
		sigx := firstMatch(parts[2], `[&?]ptsigx=([^&]+)`)
		uin := firstMatch(parts[2], `[&?]uin=([^&]+)`)
		if sigx == "" || uin == "" {
			return LoginResult{}, fmt.Errorf("登录跳转参数缺失")
		}
		if err := c.authorizeQQ(ctx, qr.client, uin, sigx); err != nil {
			return LoginResult{}, err
		}
		return LoginResult{Event: LoginDone, Message: "登录成功"}, nil
	case 66, 408:
		return LoginResult{Event: LoginWaiting, Message: "请使用手机扫码"}, nil
	case 67, 404:
		return LoginResult{Event: LoginScanned, Message: "已扫码，请在手机上确认"}, nil
	case 65, 402:
		return LoginResult{Event: LoginExpired, Message: "二维码已过期"}, nil
	case 68, 403:
		return LoginResult{Event: LoginRefused, Message: "已拒绝登录"}, nil
	default:
		return LoginResult{Event: LoginOther, Message: "等待扫码…"}, nil
	}
}

func (c *Client) authorizeQQ(ctx context.Context, hc *http.Client, uin, sigx string) error {
	values := url.Values{
		"uin":            {uin},
		"pttype":         {"1"},
		"service":        {"ptqrlogin"},
		"nodirect":       {"0"},
		"ptsigx":         {sigx},
		"s_url":          {"https://graph.qq.com/oauth2.0/login_jump"},
		"ptlang":         {"2052"},
		"ptredirect":     {"100"},
		"aid":            {"716027609"},
		"daid":           {"383"},
		"j_later":        {"0"},
		"low_login_hour": {"0"},
		"regmaster":      {"0"},
		"pt_login_type":  {"3"},
		"pt_aid":         {"0"},
		"pt_aaid":        {"16"},
		"pt_light":       {"0"},
		"pt_3rd_aid":     {"100497308"},
	}
	resp, err := get(ctx, hc, "https://ssl.ptlogin2.graph.qq.com/check_sig?"+values.Encode(), map[string]string{
		"Referer": "https://xui.ptlogin2.qq.com/",
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	pSkey := cookieValue(hc, []string{
		"https://graph.qq.com/",
		"https://ssl.ptlogin2.graph.qq.com/",
		"https://y.qq.com/",
	}, "p_skey")
	if pSkey == "" {
		return fmt.Errorf("获取 p_skey 失败")
	}

	form := url.Values{
		"response_type": {"code"},
		"client_id":     {"100497308"},
		"redirect_uri":  {"https://y.qq.com/portal/wx_redirect.html?login_type=1&surl=https://y.qq.com/"},
		"scope":         {"get_user_info,get_app_friends"},
		"state":         {"state"},
		"switch":        {""},
		"from_ptlogin":  {"1"},
		"src":           {"1"},
		"update_auth":   {"1"},
		"openapi":       {"1010_1030"},
		"g_tk":          {strconv.FormatUint(uint64(hash33(pSkey, 5381)), 10)},
		"auth_time":     {strconv.FormatInt(time.Now().UnixMilli(), 10)},
		"ui":            {uuid4()},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graph.qq.com/oauth2.0/authorize", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", defaultUA)
	req.Header.Set("Referer", "https://graph.qq.com/")
	nr := *hc
	nr.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err = nr.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("登录服务器返回 HTTP %d", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	loc, err := url.Parse(location)
	if err != nil {
		return fmt.Errorf("解析登录跳转失败")
	}
	code := loc.Query().Get("code")
	if code == "" {
		return fmt.Errorf("获取登录 code 失败")
	}

	var payload credentialPayload
	if err := c.callWith(ctx, hc, "QQConnectLogin.LoginServer", "QQLogin",
		map[string]any{"code": code}, map[string]any{"tmeLoginType": "2"}, &payload); err != nil {
		return err
	}
	return c.acceptCredential(payload, 2)
}

func (c *Client) pollWeChatQR(ctx context.Context, qr *QRLogin) (LoginResult, error) {
	values := url.Values{
		"uuid": {qr.identifier},
		"_":    {strconv.FormatInt(time.Now().UnixMilli(), 10)},
	}
	resp, err := get(ctx, qr.client, "https://lp.open.weixin.qq.com/connect/l/qrconnect?"+values.Encode(), map[string]string{
		"Referer": "https://open.weixin.qq.com/",
	})
	if err != nil {
		return LoginResult{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return LoginResult{}, err
	}
	match := regexp.MustCompile(`window\.wx_errcode=(\d+);window\.wx_code='([^']*)'`).FindSubmatch(body)
	if len(match) < 3 {
		return LoginResult{}, fmt.Errorf("获取微信二维码状态失败")
	}
	code, _ := strconv.Atoi(string(match[1]))
	switch code {
	case 405:
		wxCode := string(match[2])
		if wxCode == "" {
			return LoginResult{}, fmt.Errorf("获取微信登录 code 失败")
		}
		if err := c.authorizeWeChat(ctx, qr.client, wxCode); err != nil {
			return LoginResult{}, err
		}
		return LoginResult{Event: LoginDone, Message: "登录成功"}, nil
	case 404:
		return LoginResult{Event: LoginScanned, Message: "已扫码，请在手机上确认"}, nil
	case 408:
		return LoginResult{Event: LoginWaiting, Message: "等待扫码…"}, nil
	case 402, 65:
		return LoginResult{Event: LoginExpired, Message: "二维码已过期"}, nil
	case 403, 68:
		return LoginResult{Event: LoginRefused, Message: "已拒绝登录"}, nil
	default:
		return LoginResult{Event: LoginOther, Message: "等待扫码…"}, nil
	}
}

func (c *Client) authorizeWeChat(ctx context.Context, hc *http.Client, code string) error {
	var payload credentialPayload
	if err := c.callWith(ctx, hc, "music.login.LoginServer", "Login", map[string]any{
		"code":     code,
		"strAppid": "wx48db31d50e334801",
	}, map[string]any{"tmeLoginType": "1"}, &payload); err != nil {
		return err
	}
	return c.acceptCredential(payload, 1)
}

func (c *Client) acceptCredential(payload credentialPayload, fallbackType int) error {
	credential := payload.credential(fallbackType)
	if !credential.Valid() {
		return fmt.Errorf("登录响应缺少凭证字段")
	}
	c.setCredential(credential)
	c.setCredentialSaveError(c.SaveCredential())
	return nil
}

type credentialPayload struct {
	OpenID          string `json:"openid"`
	RefreshToken    string `json:"refresh_token"`
	AccessToken     string `json:"access_token"`
	ExpiredAt       int64  `json:"expired_at"`
	MusicID         int64  `json:"musicid"`
	Musickey        string `json:"musickey"`
	UnionID         string `json:"unionid"`
	StrMusicID      string `json:"str_musicid"`
	RefreshKey      string `json:"refresh_key"`
	EncryptUinSnake string `json:"encrypt_uin"`
	EncryptUinCamel string `json:"encryptUin"`
	LoginTypeSnake  int    `json:"login_type"`
	LoginTypeCamel  int    `json:"loginType"`
}

func (p credentialPayload) credential(fallbackType int) Credential {
	c := Credential{
		OpenID:       p.OpenID,
		RefreshToken: p.RefreshToken,
		AccessToken:  p.AccessToken,
		ExpiredAt:    p.ExpiredAt,
		MusicID:      p.MusicID,
		Musickey:     p.Musickey,
		UnionID:      p.UnionID,
		StrMusicID:   p.StrMusicID,
		RefreshKey:   p.RefreshKey,
		EncryptUin:   p.EncryptUinSnake,
		LoginType:    p.LoginTypeSnake,
	}
	if c.EncryptUin == "" {
		c.EncryptUin = p.EncryptUinCamel
	}
	if c.LoginType == 0 {
		c.LoginType = p.LoginTypeCamel
	}
	if c.LoginType == 0 {
		c.LoginType = fallbackType
	}
	if c.StrMusicID == "" {
		c.StrMusicID = strconv.FormatInt(c.MusicID, 10)
	}
	return c
}

func get(ctx context.Context, hc *http.Client, rawURL string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", defaultUA)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		resp.Body.Close()
		return nil, fmt.Errorf("登录服务器返回 HTTP %d", resp.StatusCode)
	}
	return resp, nil
}

func cookieValue(hc *http.Client, rawURLs []string, name string) string {
	if hc == nil || hc.Jar == nil {
		return ""
	}
	for _, rawURL := range rawURLs {
		u, err := url.Parse(rawURL)
		if err != nil {
			continue
		}
		for _, cookie := range hc.Jar.Cookies(u) {
			if cookie.Name == name {
				return cookie.Value
			}
		}
	}
	return ""
}

func firstMatch(s, pattern string) string {
	match := regexp.MustCompile(pattern).FindStringSubmatch(s)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func hash33(s string, h uint32) uint32 {
	for i := 0; i < len(s); i++ {
		h = h*33 + uint32(s[i])
	}
	return h & 0x7fffffff
}

func uuid4() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func randFloat() float64 {
	b := make([]byte, 7)
	_, _ = rand.Read(b)
	var n uint64
	for _, v := range b {
		n = n<<8 | uint64(v)
	}
	return float64(n) / float64(uint64(1)<<56)
}

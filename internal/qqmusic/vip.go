package qqmusic

import (
	"context"
	"errors"
	"strings"
)

// VipInfo is the signed-in account's membership summary.
type VipInfo struct {
	SuperVIP bool
	GreenVip bool
	Level    int
	Expiry   string
}

// VipInfo reads the signed-in account's membership status and expiry.
func (c *Client) VipInfo(ctx context.Context) (VipInfo, error) {
	if !c.HasCredential() {
		return VipInfo{}, errors.New("需要登录后才能获取会员信息")
	}
	var data struct {
		SVip     int    `json:"svip"`
		Send     string `json:"send"`
		Identity struct {
			Vip      int    `json:"vip"`
			Level    int    `json:"level"`
			Overdate string `json:"overdate"`
		} `json:"identity"`
	}
	if err := c.call(ctx, "VipLogin.VipLoginInter", "vip_login_base", map[string]any{}, &data); err != nil {
		return VipInfo{}, err
	}
	info := VipInfo{
		SuperVIP: data.SVip != 0,
		GreenVip: data.Identity.Vip != 0,
		Level:    data.Identity.Level,
	}
	expiry := data.Identity.Overdate
	if info.SuperVIP && strings.TrimSpace(data.Send) != "" {
		expiry = data.Send
	}
	info.Expiry = dateOnly(expiry)
	return info, nil
}

func dateOnly(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 10 {
		value = value[:10]
	}
	return value
}

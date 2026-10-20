package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/rzh0504/qmdr/internal/qqmusic"
)

func friendlyError(err error) string {
	if err == nil {
		return ""
	}
	var apiErr *qqmusic.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case 1000:
			return "登录已过期，请重新登录"
		case 2000:
			return "请求校验失败，请稍后重试"
		}
		if apiErr.Message != "" {
			return apiErr.Message
		}
		return fmt.Sprintf("接口错误（%d）", apiErr.Code)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "请求超时，请检查网络"
	}
	return err.Error()
}

package apperr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/rzh0504/qmdr/internal/qqmusic"
)

// Friendly turns an error into a short message for the UI.
func Friendly(err error) string {
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
	if errors.Is(err, context.Canceled) {
		return "请求已取消"
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return "下载中断，请重试"
	}
	if errors.Is(err, os.ErrPermission) {
		return "没有访问文件或目录的权限"
	}
	if errors.Is(err, os.ErrNotExist) {
		return "文件或目录不存在"
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "网络超时，请稍后重试"
		}
		return "网络连接失败，请检查网络"
	}
	return err.Error()
}

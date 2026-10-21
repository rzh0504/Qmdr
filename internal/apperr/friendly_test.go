package apperr

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"testing"

	"github.com/rzh0504/qmdr/internal/qqmusic"
)

func TestFriendly(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{context.Canceled, "请求已取消"},
		{context.DeadlineExceeded, "请求超时，请检查网络"},
		{io.ErrUnexpectedEOF, "下载中断，请重试"},
		{&os.PathError{Op: "open", Path: "test", Err: os.ErrPermission}, "没有访问文件或目录的权限"},
		{&os.PathError{Op: "open", Path: "test", Err: os.ErrNotExist}, "文件或目录不存在"},
		{&net.OpError{Op: "dial", Net: "tcp", Err: errors.New("boom")}, "网络连接失败，请检查网络"},
		{&qqmusic.APIError{Code: 1000}, "登录已过期，请重新登录"},
		{errors.New("plain"), "plain"},
	}
	for _, tc := range cases {
		if got := Friendly(tc.err); got != tc.want {
			t.Errorf("Friendly(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

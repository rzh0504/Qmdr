package main

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"testing"
)

func TestFriendlyError(t *testing.T) {
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
		{errors.New("plain"), "plain"},
	}
	for _, tc := range cases {
		if got := friendlyError(tc.err); got != tc.want {
			t.Errorf("friendlyError(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

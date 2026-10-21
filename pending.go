package main

import "github.com/rzh0504/qmdr/internal/pending"

func (a *app) savePending() {
	path := pending.Path()
	if path == "" {
		a.setToast("找不到配置目录")
		return
	}
	songs := pending.Normalize(a.queue.pending)
	a.dispatchDisk(func() {
		if err := pending.SaveFile(path, songs); err != nil {
			a.update(func() {
				a.toast = "待下载队列保存失败：" + friendlyError(err)
			})
		}
	})
}

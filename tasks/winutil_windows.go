//go:build windows

package tasks

import (
	"syscall"
	"unsafe"
)

// 单实例激活已有窗口所需的最小窗口工具（从原 dwm_windows.go 提取，
// Mica/WebView2 相关代码已随 mygo 改写删除）。

var (
	user32                 = syscall.NewLazyDLL("user32.dll")
	procFindWindowW        = user32.NewProc("FindWindowW")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
)

// findWindowByTitle 通过窗口标题查找窗口句柄。
func findWindowByTitle(title string) uintptr {
	titleUTF16, _ := syscall.UTF16PtrFromString(title)
	ret, _, _ := procFindWindowW.Call(
		0,
		uintptr(unsafe.Pointer(titleUTF16)),
	)
	return ret
}

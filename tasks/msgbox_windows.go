//go:build windows

package tasks

import (
	"syscall"
	"unsafe"
)

var (
	user32Lib        = syscall.NewLazyDLL("user32.dll")
	procMessageBoxW  = user32Lib.NewProc("MessageBoxW")
)

const (
	mbIconError   = 0x00000010
	mbIconWarning = 0x00000030
	mbSetForeground = 0x00010000
)

// showMessageBox 弹出系统对话框（GUI 程序 stderr 不可见，出错必须弹窗告知用户）
func showMessageBox(title, text string, isError bool) {
	flags := uintptr(mbSetForeground | mbIconWarning)
	if isError {
		flags = mbSetForeground | mbIconError
	}
	t, _ := syscall.UTF16PtrFromString(title)
	x, _ := syscall.UTF16PtrFromString(text)
	procMessageBoxW.Call(0,
		uintptr(unsafe.Pointer(x)),
		uintptr(unsafe.Pointer(t)),
		flags)
}

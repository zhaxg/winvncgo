//go:build windows

package tasks

import (
	"syscall"
	"unsafe"
)

var (
	dwmapi                  = syscall.NewLazyDLL("dwmapi.dll")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	procDwmExtendFrame       = dwmapi.NewProc("DwmExtendFrameIntoClientArea")

	user32              = syscall.NewLazyDLL("user32.dll")
	procFindWindowW         = user32.NewProc("FindWindowW")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
)

const (
	DWMWA_USE_IMMERSIVE_DARK_MODE = 20
	DWMWA_SYSTEMBACKDROP_TYPE     = 38

	DWMSBT_NONE             = 0
	DWMSBT_MAINWINDOW       = 2 // Mica
	DWMSBT_TRANSIENTWINDOW  = 3 // Acrylic
	DWMSBT_TABBEDWINDOW     = 4 // Mica Alt
)

type _MARGINS struct {
	Left, Right, Top, Bottom int32
}

// applyBackdrop 为指定窗口设置 DWM 背景效果
// backdropType: 2=Mica, 3=Acrylic, 4=Mica Alt
func applyBackdrop(hwnd uintptr, backdropType int) {
	// 设置深色模式（可选，让 Mica 效果更好看）
	darkMode := int32(1)
	procDwmSetWindowAttribute.Call(
		hwnd,
		DWMWA_USE_IMMERSIVE_DARK_MODE,
		uintptr(unsafe.Pointer(&darkMode)),
		unsafe.Sizeof(darkMode),
	)

	// 设置背景类型
	val := int32(backdropType)
	procDwmSetWindowAttribute.Call(
		hwnd,
		DWMWA_SYSTEMBACKDROP_TYPE,
		uintptr(unsafe.Pointer(&val)),
		unsafe.Sizeof(val),
	)

	// 扩展边框到客户区，让 Mica 效果覆盖整个窗口
	margins := _MARGINS{Left: -1, Right: -1, Top: -1, Bottom: -1}
	procDwmExtendFrame.Call(
		hwnd,
		uintptr(unsafe.Pointer(&margins)),
	)
}

// findWindowByTitle 通过窗口标题查找窗口句柄
func findWindowByTitle(title string) uintptr {
	titleUTF16, _ := syscall.UTF16PtrFromString(title)
	ret, _, _ := procFindWindowW.Call(
		0,
		uintptr(unsafe.Pointer(titleUTF16)),
	)
	return ret
}

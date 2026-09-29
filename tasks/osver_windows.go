//go:build windows

package tasks

import (
	"syscall"
	"unsafe"
)

var (
	ntdll             = syscall.NewLazyDLL("ntdll.dll")
	procRtlGetVersion = ntdll.NewProc("RtlGetVersion")
)

type osVersionInfoEx struct {
	OSVersionInfoSize uint32
	MajorVersion      uint32
	MinorVersion      uint32
	BuildNumber       uint32
	PlatformId        uint32
	CSDVersion        [128]uint16
	ServicePackMajor  uint16
	ServicePackMinor  uint16
	SuiteMask         uint16
	ProductType       byte
	Reserved          byte
}

// osVersion 返回真实的 Windows 版本号。
// RtlGetVersion 不受应用清单里的 supportedOS 影响，
// 不会像 GetVersionEx 那样在 Win8+ 被兼容性虚报成 6.2。
func osVersion() (major, minor, build uint32) {
	var info osVersionInfoEx
	info.OSVersionInfoSize = uint32(unsafe.Sizeof(info))
	ret, _, _ := procRtlGetVersion.Call(uintptr(unsafe.Pointer(&info)))
	if ret != 0 { // STATUS_SUCCESS == 0
		return 0, 0, 0
	}
	return info.MajorVersion, info.MinorVersion, info.BuildNumber
}

// isMicaSupported 仅 Windows 11（build >= 22000）支持 Mica 系统背景。
// Win7/8/10 上 DwmSetWindowAttribute(DWMWA_SYSTEMBACKDROP_TYPE) 无效，
// 而把窗口背景设成透明会产生异常视觉效果，因此整段效果直接跳过。
func isMicaSupported() bool {
	major, _, build := osVersion()
	return major >= 10 && build >= 22000
}

//go:build !windows

package tasks

// osVersion 非 Windows 平台占位
func osVersion() (major, minor, build uint32) { return 0, 0, 0 }

// isMicaSupported 非 Windows 平台占位（Mica 为 Windows 11 特性）
func isMicaSupported() bool { return false }

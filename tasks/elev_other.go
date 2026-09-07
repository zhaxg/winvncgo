//go:build !windows

package tasks

// isAdmin 非 Windows 平台占位（无需提权检测）
func isAdmin() bool { return true }

// warnIfNotElevated 非 Windows 平台占位
func warnIfNotElevated(logger interface{ log(string) }) {}

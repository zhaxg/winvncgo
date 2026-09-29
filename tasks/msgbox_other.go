//go:build !windows

package tasks

// showMessageBox 非 Windows 平台占位（只输出日志）
func showMessageBox(title, text string, isError bool) {}

//go:build !windows

package tasks

// resolveWebView2BrowserPath 非 Windows 平台占位（返回空 = 使用默认运行时）
func resolveWebView2BrowserPath(baseDir string) string { return "" }

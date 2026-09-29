//go:build windows

package tasks

import (
	"path/filepath"

	"github.com/wailsapp/go-webview2/webviewloader"
)

// resolveWebView2BrowserPath 决定传给 Wails 的 WebView2 运行时目录（WebviewBrowserPath）。
//
// Windows 7/8/8.1：官方支持止步于 109.x，且微软在线安装器已不可用，
// 依赖随程序分发的固定版本运行时（webview2\ 目录）。
// Windows 10+：优先用系统安装的 WebView2，返回空字符串即可。
//
// 注意：Wails v2.15 会清空 WEBVIEW2_BROWSER_EXECUTABLE_FOLDER 环境变量，
// 环境变量方式无效，只能走这里的 WebviewBrowserPath 选项。
func resolveWebView2BrowserPath(baseDir string) string {
	major, _, _ := osVersion()
	if major == 0 || major >= 10 {
		return "" // Win10+ 或未知系统：用系统 WebView2
	}

	// Win7/8/8.1
	if p := findFixedWebView2Runtime(baseDir); p != "" {
		return p
	}
	// 没有固定版本运行时，但系统已装过 109.x 也能用
	if _, err := webviewloader.GetAvailableCoreWebView2BrowserVersionString(""); err == nil {
		return ""
	}
	// 都没有：指向约定目录（即使不存在），让 Wails 报出明确的中文提示，
	// 而不是走在线安装 WebView2 的流程（Win7 上必然失败）
	return filepath.Join(baseDir, "webview2")
}

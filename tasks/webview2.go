package tasks

import (
	"os"
	"path/filepath"
	"strings"
)

// findFixedWebView2Runtime 在程序目录下查找固定版本（Fixed Version）WebView2 运行时。
//
// Windows 7 / 8 / 8.1 上 WebView2 官方支持止步于 109.x（2023 年 1 月），
// 微软的在线安装器已无法在这些系统上安装，只能以固定版本形式随程序分发：
// 将运行时目录放在程序目录下的 webview2\，
// 或命名为 Microsoft.WebView2.FixedVersionRuntime.<版本>_x64\。
//
// 返回找到的目录；找不到返回空字符串。
func findFixedWebView2Runtime(baseDir string) string {
	// 合法的固定版本运行时目录：根下有 msedgewebview2.exe，
	// 且 EBWebView\<arch>\EmbeddedBrowserWebView.dll 存在（go-webview2 靠它加载）
	hasRuntime := func(dir string) bool {
		if dir == "" {
			return false
		}
		if st, err := os.Stat(filepath.Join(dir, "msedgewebview2.exe")); err != nil || st.IsDir() {
			return false
		}
		entries, err := os.ReadDir(filepath.Join(dir, "EBWebView"))
		if err != nil {
			return false
		}
		for _, e := range entries {
			if e.IsDir() {
				p := filepath.Join(dir, "EBWebView", e.Name(), "EmbeddedBrowserWebView.dll")
				if st, err := os.Stat(p); err == nil && !st.IsDir() {
					return true
				}
			}
		}
		return false
	}

	// 1. 约定目录 webview2\
	if p := filepath.Join(baseDir, "webview2"); hasRuntime(p) {
		return p
	}

	// 2. 微软固定版本分发的默认目录名 Microsoft.WebView2.FixedVersionRuntime.*
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "Microsoft.WebView2.FixedVersionRuntime.") {
			continue
		}
		if p := filepath.Join(baseDir, e.Name()); hasRuntime(p) {
			return p
		}
	}
	return ""
}

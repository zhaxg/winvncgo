package tasks

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	winopts "github.com/wailsapp/wails/v2/pkg/options/windows"
)

// Run 启动 Wails 应用，assets 由调用方提供
func Run(assets fs.FS) {
	if !acquireInstanceMutex() {
		activateExistingWindow()
		os.Exit(0)
	}

	winOptions := &winopts.Options{
		Messages: &winopts.Messages{
			InvalidFixedWebview2: "WebView2 运行时无效。Windows 7 请把固定版本运行时目录（内含 msedgewebview2.exe 和 EBWebView\\）" +
				"放到程序目录的 webview2\\ 下，或安装 109.x 版本的 WebView2 运行时后重试。",
		},
	}

	// Windows 7/8/8.1 上 WebView2 官方支持止步于 109.x，微软在线安装器已不可用，
	// 需要随程序分发固定版本运行时（webview2\ 目录）。
	if browserPath := resolveWebView2BrowserPath(executableDir()); browserPath != "" {
		log.Println("使用 WebView2 运行时目录:", browserPath)
		winOptions.WebviewBrowserPath = browserPath
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:         "远程协助",
		Width:         420,
		Height:        570,
		DisableResize: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Windows:    winOptions,
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatal("Error: ", err.Error())
	}
}

// executableDir 返回可执行文件所在目录（失败时退回工作目录）
func executableDir() string {
	if exePath, err := os.Executable(); err == nil {
		return filepath.Dir(exePath)
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

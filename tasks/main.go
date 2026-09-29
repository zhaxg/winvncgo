package tasks

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	winopts "github.com/wailsapp/wails/v2/pkg/options/windows"
)

// Run 启动 Wails 应用，assets 由调用方提供
func Run(assets fs.FS) {
	// 尽早打开日志：GUI 程序没有控制台，启动失败如果只写 stderr 将毫无痕迹
	baseDir := executableDir()
	openAppLog(baseDir)
	defer closeAppLog()
	logStartupEnv(baseDir)

	// 兜底捕获 panic，写日志并弹窗，避免进程无声无息消失
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			appLogf("!!! panic: %v\n%s", r, stack)
			showMessageBox("远程协助 - 启动失败",
				"程序发生内部错误，已写入日志 winvncgo.log：\n\n"+
					toShortText(r), true)
			os.Exit(1)
		}
	}()

	if !acquireInstanceMutex() {
		appLog("已有实例在运行，激活已有窗口后退出")
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
	browserPath := resolveWebView2BrowserPath(baseDir)
	if browserPath != "" {
		appLogf("使用 WebView2 运行时目录: %s", browserPath)
		winOptions.WebviewBrowserPath = browserPath
	} else {
		appLog("使用系统安装的 WebView2 运行时")
	}

	app := NewApp()
	appLog("初始化完成，进入 wails.Run")

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
		// WebView2 缺失/无效、窗口创建失败等都会走到这里；
		// 不弹窗的话用户只会看到“程序打不开”，完全无从排查
		appLogf("wails.Run 失败: %v", err)
		showMessageBox("远程协助 - 启动失败",
			"界面启动失败：\n\n"+err.Error(), true)
		log.Fatal("Error: ", err.Error())
	}
	appLog("wails.Run 正常结束")
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

// toShortText 把 recover 到的内容转成短文本用于弹窗
func toShortText(r interface{}) string {
	s := toString(r)
	if len(s) > 400 {
		s = s[:400] + "..."
	}
	return s
}

func toString(r interface{}) string {
	if s, ok := r.(string); ok {
		return s
	}
	if e, ok := r.(error); ok {
		return e.Error()
	}
	return "unknown"
}

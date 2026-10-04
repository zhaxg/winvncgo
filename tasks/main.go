package tasks

import (
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Run 启动 mygo 版应用：native UI 窗口，无 WebView2 依赖。
//
// 与 Wails 版一致的启动保障：文件日志 winvncgo.log、panic 兜底、
// 单实例互斥（第二个实例激活已有窗口后退出）。
func Run() {
	baseDir := executableDir()
	openAppLog(baseDir)
	defer closeAppLog()
	logStartupEnv(baseDir)

	// 兜底 panic：写进日志，弹窗提示（GUI 程序没有控制台，
	// 只写 stderr 的日志无处可看）
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			appLogf("!!! panic: %v\n%s", r, stack)
			showMessageBox("远程协助 - 启动失败",
				"程序出现问题，详见日志 winvncgo.log。\n\n"+
					toShortText(r), true)
			os.Exit(1)
		}
	}()

	if !acquireInstanceMutex() {
		appLog("已有实例在运行，激活已有窗口后退出")
		activateExistingWindow()
		os.Exit(0)
	}

	// 内嵌字体：Win10 1703+ 直接注册；Win7 走按用户安装降级
	ensureMapleMono()

	app := NewApp()
	appLogf("初始化完成，基础目录: %s，进入 mygo.App.Run", app.baseDir)

	mygo.App.WhenReady(func() {
		u := newUI(app)
		win := mygo.NewWindow(mygo.WindowOptions{
			Title: "远程协助",
			Width: 420, Height: 570,
			// 与 Wails 版 DisableResize 一致：固定窗口大小
			MinWidth: 420, MaxWidth: 420,
			MinHeight: 570, MaxHeight: 570,
			Content: ui.View(u.view),
		})
		u.win = win
		if err := win.SetIcon(appIconPNG); err != nil {
			appLogf("窗口图标设置失败: %v", err)
		}
		appLog("窗口已创建")

		// 界面就绪后注册状态回调，再启动后端循环，
		// 保证回调触发时 u.win 已赋值（happens-before 由 goroutine 创建保证）
		app.onStatus = u.onStatus
		app.startup()
		u.setupTray(win)
		appLog("启动完成")
	})

	if err := mygo.App.Run(); err != nil {
		appLogf("mygo.App.Run 失败: %v", err)
		showMessageBox("远程协助 - 启动失败", err.Error(), true)
		os.Exit(1)
	}
	app.shutdown()
}

// executableDir 返回可执行文件所在目录，失败时回退到工作目录。
func executableDir() string {
	if exePath, err := os.Executable(); err == nil {
		return filepath.Dir(exePath)
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

// toShortText 把 recover 的异常转换成短文本。
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

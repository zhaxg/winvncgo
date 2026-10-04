package tasks

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

// App 是应用后端：管理 VNC 服务、控制中心连接与设备 ID。
// mygo 版本不再依赖 Wails：状态通过 onStatus 回调推给界面，
// 由界面用 Window.Update 在主线程上刷新。
type App struct {
	ctx     context.Context
	cancel  context.CancelFunc
	mgr     *Manager
	baseDir string

	// onStatus 由界面在启动状态循环之前注册，每 2 秒推送一次状态。
	onStatus func(StatusEvent)
}

// NewApp 创建 App 应用结构体。
func NewApp() *App {
	exePath, _ := os.Executable()
	baseDir := filepath.Dir(exePath)

	if !fileExists(filepath.Join(baseDir, "winvnc.exe")) {
		if wd, err := os.Getwd(); err == nil && fileExists(filepath.Join(wd, "winvnc.exe")) {
			baseDir = wd
		}
	}

	cfg := loadConfig(filepath.Join(baseDir, "winvncgo.json"))
	mgr := newManager(cfg, baseDir)

	return &App{
		mgr:     mgr,
		baseDir: baseDir,
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// startup 初始化管理器并启动状态循环与 ID 轮换循环。
// 需要在界面就绪之后、注册好 onStatus 之后调用（主线程）。
func (a *App) startup() {
	a.ctx, a.cancel = context.WithCancel(context.Background())
	a.mgr.Initialize()
	go a.statusLoop()
	go a.mgr.rotationLoop()
}

// statusLoop 定时推送状态给界面（VNC 服务停止时自动拉起）。
func (a *App) statusLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// VNC 服务停止时自动拉起
			if !a.mgr.vnc.IsRunning() {
				a.mgr.vnc.EnsureService()
			}
			if a.onStatus != nil {
				a.onStatus(a.mgr.currentStatus())
			}
		case <-a.ctx.Done():
			return
		}
	}
}

// GetStatus 获取当前状态（启动时的初始填充）。
func (a *App) GetStatus() map[string]interface{} {
	return a.mgr.GetStatus()
}

// RefreshID 刷新设备 ID。
func (a *App) RefreshID() map[string]interface{} {
	err := a.mgr.RefreshID()
	if err != nil {
		return map[string]interface{}{"ok": false, "error": err.Error()}
	}
	return map[string]interface{}{"ok": true, "id": a.mgr.CopyID()}
}

// Connect 连接远程设备。
func (a *App) Connect(id, ip string) map[string]interface{} {
	err := a.mgr.Connect(id, ip)
	if err != nil {
		return map[string]interface{}{"ok": false, "error": err.Error()}
	}
	return map[string]interface{}{"ok": true}
}

// CopyID 复制当前 ID。
func (a *App) CopyID() string {
	return a.mgr.CopyID()
}

// shutdown 应用退出时调用：停状态循环、停 VNC 服务。
func (a *App) shutdown() {
	if a.cancel != nil {
		a.cancel()
	}
	a.mgr.Shutdown()
	appLog("应用已关闭")
}

// GetBaseDir 获取基础目录。
func (a *App) GetBaseDir() string {
	return a.baseDir
}

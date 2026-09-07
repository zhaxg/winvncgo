package tasks

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx    context.Context
	mgr    *Manager
	baseDir string
}

// NewApp creates a new App application struct
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

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.mgr.Initialize()
	go a.statusLoop()
	go a.mgr.rotationLoop()

	// 应用 Windows 11 Mica 效果
	a.applyMica()
}

// statusLoop 定时推送状态到前端
func (a *App) statusLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			status := a.mgr.currentStatus()
			runtime.EventsEmit(a.ctx, "status", status)
		case <-a.ctx.Done():
			return
		}
	}
}

// GetStatus 获取当前状态
func (a *App) GetStatus() map[string]interface{} {
	return a.mgr.GetStatus()
}

// RefreshID 刷新设备 ID
func (a *App) RefreshID() map[string]interface{} {
	err := a.mgr.RefreshID()
	if err != nil {
		return map[string]interface{}{"ok": false, "error": err.Error()}
	}
	return map[string]interface{}{"ok": true, "id": a.mgr.CopyID()}
}

// Connect 连接远程设备
func (a *App) Connect(id, ip string) map[string]interface{} {
	err := a.mgr.Connect(id, ip)
	if err != nil {
		return map[string]interface{}{"ok": false, "error": err.Error()}
	}
	return map[string]interface{}{"ok": true}
}

// CopyID 复制当前 ID
func (a *App) CopyID() string {
	return a.mgr.CopyID()
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	a.mgr.Shutdown()
	log.Println("应用已关闭")
}

// GetBaseDir 获取基础目录
func (a *App) GetBaseDir() string {
	return a.baseDir
}

// applyMica 应用 Windows 11 Mica 背景效果
func (a *App) applyMica() {
	// 设置窗口背景为透明，让 DWM 效果透过来
	runtime.WindowSetBackgroundColour(a.ctx, 0, 0, 0, 0)

	// 通过窗口标题查找 HWND，然后设置 Mica 效果
	hwnd := findWindowByTitle("远程协助")
	if hwnd != 0 {
		applyBackdrop(hwnd, DWMSBT_MAINWINDOW) // Mica
	}
}

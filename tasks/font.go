package tasks

// Maple Mono 字体接入：
//   - Windows 10 1703+ / macOS / Linux：ui.RegisterFont 直接注册内嵌字体；
//   - Windows 7：mygo 的 RegisterFont 依赖 IDWriteFactory5（Win10 1703+），
//     会返回错误。此时降级为「按用户安装」——把 TTF 复制到
//     %LOCALAPPDATA%\Microsoft\Windows\Fonts 并写 HKCU 注册表（无需管理员），
//     DirectWrite 的系统字体集合即可枚举到它；系统已装过则直接用系统里的。
//   两条路都失败也不退出：主题字体族列表
//   "Maple Mono, Segoe UI, Microsoft YaHei" 会自动回退到系统字体。

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"

	"github.com/egoist/mygo/ui"
)

//go:embed fonts/maple-mono.ttf
var mapleMonoTTF []byte // 由原前端 woff2 转换而来

func ensureMapleMono() {
	if err := ui.RegisterFont(mapleMonoTTF, "Maple Mono"); err != nil {
		appLogf("RegisterFont 失败（Win10 1703 以下属预期）: %v", err)
		installMapleMonoPerUser()
		return
	}
	appLogf("RegisterFont ok: Maple Mono (%d bytes)", len(mapleMonoTTF))
}

// installMapleMonoPerUser 把内嵌 TTF 安装到当前用户的字体目录并注册，
// 全程写 HKCU，不需要管理员权限。系统或当前用户已装过时跳过。
func installMapleMonoPerUser() {
	if mapleMonoInstalled() {
		appLog("font: 系统/用户已安装 Maple Mono，跳过安装，直接使用系统字体集合中的它")
		return
	}
	const fileName = "MapleMono-Regular.ttf"
	dir := filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Windows", "Fonts")
	dest := filepath.Join(dir, fileName)

	if _, err := os.Stat(dest); err == nil {
		appLogf("font: 文件已存在 %s", dest)
	} else {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			appLogf("font: 创建目录失败 %s: %v", dir, err)
			return
		}
		if err := os.WriteFile(dest, mapleMonoTTF, 0o644); err != nil {
			appLogf("font: 写入失败 %s: %v", dest, err)
			return
		}
		appLogf("font: 已安装文件 %s (%d bytes)", dest, len(mapleMonoTTF))
	}

	// 注册：HKCU\Software\Microsoft\Windows NT\CurrentVersion\Fonts
	key, _, err := registry.CreateKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows NT\CurrentVersion\Fonts`, registry.SET_VALUE)
	if err != nil {
		appLogf("font: 打开 HKCU Fonts 键失败: %v", err)
		return
	}
	defer key.Close()
	if err := key.SetStringValue("Maple Mono (TrueType)", dest); err != nil {
		appLogf("font: 写 HKCU 注册表失败: %v", err)
		return
	}
	appLogf("font: 已注册 HKCU Fonts = %s", dest)

	// 广播 WM_FONTCHANGE，通知已运行的应用字体集合有变
	const (
		hwndBroadcast = 0xFFFF
		wmFontChange  = 0x001D
		smtoAbortHung = 0x2
	)
	var result uintptr
	r1, _, _ := syscall.NewLazyDLL("user32.dll").NewProc("SendMessageTimeoutW").Call(
		hwndBroadcast, wmFontChange, 0, 0, smtoAbortHung, 2000,
		uintptr(unsafe.Pointer(&result)))
	appLogf("font: WM_FONTCHANGE 广播 r1=%d", r1)
}

// mapleMonoInstalled 查机器级（HKLM）与用户级（HKCU）字体注册表，
// 判断 Maple Mono 是否已安装。
func mapleMonoInstalled() bool {
	keys := []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER}
	for _, root := range keys {
		key, err := registry.OpenKey(root,
			`Software\Microsoft\Windows NT\CurrentVersion\Fonts`, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		names, err := key.ReadValueNames(-1)
		key.Close()
		if err != nil {
			continue
		}
		for _, n := range names {
			// 值名形如 "Maple Mono (TrueType)" / "MapleMono Regular (TrueType)"
			norm := strings.ToLower(strings.ReplaceAll(n, " ", ""))
			if strings.Contains(norm, "maplemono") {
				return true
			}
		}
	}
	return false
}

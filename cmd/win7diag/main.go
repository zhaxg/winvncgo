// win7diag: Windows 7 兼容性诊断工具（控制台程序，仅依赖 Go 标准库）。
//
// 目的：winvncgo.exe 是 GUI 程序，启动失败时 stderr 无处显示，进程可能一闪而过。
// 本工具用同一套 Go 工具链编译，但保留控制台：如果它能打印出下面的检测结果，
// 说明 Go 运行时本身没问题，问题在 winvncgo.exe 的 Wails/WebView2 阶段；
// 如果它启动就报 "fatal error: xxx not found"（或压根起不来），
// 则是 Go 运行时/系统补丁层面的问题（如缺 KB2533623），需要另行处理。
//
// 用法：把 win7diag.exe 放到 winvncgo.exe 同目录，在命令行里运行：
//   win7diag.exe
// 结果同时打印到屏幕并写入 win7diag.log，请把该文件发回分析。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	ntdll    = syscall.NewLazyDLL("ntdll.dll")
)

const loadLibrarySearchSystem32 = 0x00000800 // 需要 Win7 的 KB2533623

type osVersionInfoEx struct {
	OSVersionInfoSize uint32
	MajorVersion      uint32
	MinorVersion      uint32
	BuildNumber       uint32
	PlatformId        uint32
	CSDVersion        [128]uint16
	ServicePackMajor  uint16
	ServicePackMinor  uint16
	SuiteMask         uint16
	ProductType       byte
	Reserved          byte
}

var out *os.File

func p(format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	fmt.Print(s)
	if out != nil {
		out.WriteString(s)
	}
}

func main() {
	noPause := false
	for _, a := range os.Args[1:] {
		if a == "-nopause" {
			noPause = true
		}
	}
	defer func() {
		if r := recover(); r != nil {
			p("\n[!] 发生 panic: %v\n", r)
		}
		if !noPause {
			p("\n(诊断结束，结果也已写入 win7diag.log，按回车关闭窗口...)\n")
			fmt.Scanln()
		}
	}()

	baseDir := "."
	if exe, err := os.Executable(); err == nil {
		baseDir = filepath.Dir(exe)
	}
	logPath := filepath.Join(baseDir, "win7diag.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		tmp := filepath.Join(os.TempDir(), "win7diag.log")
		f, _ = os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		logPath = tmp
	}
	out = f
	defer out.Close()

	p("===== WinVNC Go Win7 诊断 =====\n")
	p("时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	p("日志: %s\n\n", logPath)

	// 1. Go 运行时已初始化 —— 能走到这里就说明运行时没在 init 阶段挂掉
	p("[1] Go 运行时: %s %s/%s —— 运行时初始化成功\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH)

	// 2. 系统版本
	var vi osVersionInfoEx
	vi.OSVersionInfoSize = uint32(unsafe.Sizeof(vi))
	procRtl := ntdll.NewProc("RtlGetVersion")
	ret, _, _ := procRtl.Call(uintptr(unsafe.Pointer(&vi)))
	if ret == 0 {
		p("[2] 系统版本: Windows %d.%d build %d (SP%d.%d)\n",
			vi.MajorVersion, vi.MinorVersion, vi.BuildNumber,
			vi.ServicePackMajor, vi.ServicePackMinor)
	} else {
		p("[2] 系统版本: RtlGetVersion 调用失败\n")
	}

	// 3. 32/64 位系统（64 位程序跑在 32 位系统上根本起不来）
	is64 := runtime.GOARCH == "amd64" && unsafe.Sizeof(uintptr(0)) == 8
	procIsWow64 := kernel32.NewProc("IsWow64Process")
	var wow64 bool
	if procIsWow64.Find() == nil {
		h, _, _ := kernel32.NewProc("GetCurrentProcess").Call()
		r, _, _ := procIsWow64.Call(h, uintptr(unsafe.Pointer(&wow64)))
		if r != 0 {
			p("[3] 进程架构: %s, WOW64=%v（WOW64=true 表示 32 位进程跑在 64 位系统）\n",
				runtime.GOARCH, wow64)
		}
	}
	if runtime.GOARCH == "386" {
		p("    注意: 本工具是 32 位版。winvncgo.exe 是 64 位，只能在 64 位 Windows 7 上运行。\n")
	}
	_ = is64

	// 4. LoadLibraryExW + LOAD_LIBRARY_SEARCH_SYSTEM32（Go 运行时启动时就会这么干；
	//    Win7 缺 KB2533623 时该标志不受支持，加载直接失败，Go 运行时会 throw 并无声退出）
	procLoadEx := kernel32.NewProc("LoadLibraryExW")
	procGetProc := kernel32.NewProc("GetProcAddress")
	procFree := kernel32.NewProc("FreeLibrary")

	loadTest := func(name string, flags uintptr) uintptr {
		p16, _ := syscall.UTF16PtrFromString(name)
		h, _, _ := procLoadEx.Call(uintptr(unsafe.Pointer(p16)), 0, flags)
		return h // 0 = 失败
	}

	name := "bcryptprimitives.dll"
	h := loadTest(name, loadLibrarySearchSystem32)
	if h == 0 {
		p("[4] LoadLibraryExW(%s, LOAD_LIBRARY_SEARCH_SYSTEM32) 失败  <-- 可疑！\n", name)
		p("    该标志需要 Windows 7 的 KB2533623（2011 年安全更新）。缺失时 Go 程序启动即崩溃。\n")
		p("    请安装 KB2533623 / KB2999226 后重试；或改用兼容构建。\n")
		h2 := loadTest(name, 0)
		if h2 != 0 {
			p("    （不带该标志可以加载成功，进一步印证是补丁缺失）\n")
			procFree.Call(h2)
		}
	} else {
		p("[4] LoadLibraryExW(%s, LOAD_LIBRARY_SEARCH_SYSTEM32) 成功\n", name)
		pn, _ := syscall.BytePtrFromString("ProcessPrng")
		fp, _, _ := procGetProc.Call(h, uintptr(unsafe.Pointer(pn)))
		if fp == 0 {
			p("    [!] 但缺少 ProcessPrng 导出（Go 运行时必需）\n")
		} else {
			p("    ProcessPrng 导出存在（Go 运行时随机数依赖，正常）\n")
		}
		procFree.Call(h)
	}

	// 5. kernel32 关键导出（Go 运行时静态依赖）
	p("[5] kernel32 关键导出:\n")
	for _, fn := range []string{
		"CreateWaitableTimerExW",      // Win7+
		"GetQueuedCompletionStatusEx", // Win7+
		"RaiseFailFastException",      // Win7+
		"GetErrorMode",                // Vista+
		"WerGetFlags",                 // Vista+
	} {
		if kernel32.NewProc(fn).Find() == nil {
			p("    %s: OK\n", fn)
		} else {
			p("    %s: 缺失  <-- 异常\n", fn)
		}
	}

	// 6. 管理员权限（winvnc 捕获屏幕需要）
	p("[6] 管理员权限: %v（winvnc 捕获屏幕需要管理员）\n", isAdminUser())

	// 7. WebView2 运行时检测
	p("[7] WebView2 运行时:\n")
	webview2Check(baseDir)

	// 8. 目录文件
	p("[8] 当前目录 (%s):\n", baseDir)
	for _, fn := range []string{"winvncgo.exe", "winvnc.exe", "vncviewer.exe", "winvncgo.json", "ultravnc.ini"} {
		if st, err := os.Stat(filepath.Join(baseDir, fn)); err == nil {
			p("    %s: %d 字节\n", fn, st.Size())
		} else {
			p("    %s: 不存在\n", fn)
		}
	}
	if st, err := os.Stat(filepath.Join(baseDir, "webview2", "msedgewebview2.exe")); err == nil {
		p("    webview2\\msedgewebview2.exe: %d 字节\n", st.Size())
	} else {
		p("    webview2\\msedgewebview2.exe: 不存在\n")
	}
}

func webview2Check(baseDir string) {
	k32 := kernel32.NewProc("GetModuleHandleW")
	_ = k32

	// 注册表: HKLM\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}
	advapi := syscall.NewLazyDLL("advapi32.dll")
	procOpen := advapi.NewProc("RegOpenKeyExW")
	procQuery := advapi.NewProc("RegQueryValueExW")
	procClose := advapi.NewProc("RegCloseKey")

	paths := []string{
		`SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`,
		`SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`,
	}
	const hklm = 0x80000002
	found := false
	for _, rp := range paths {
		rp16, _ := syscall.UTF16PtrFromString(rp)
		var hKey uintptr
		r, _, _ := procOpen.Call(hklm, uintptr(unsafe.Pointer(rp16)), 0, 0x20019, uintptr(unsafe.Pointer(&hKey)))
		if r != 0 {
			continue
		}
		pvName, _ := syscall.UTF16PtrFromString("pv")
		var buf [64]uint16
		var n uint32 = 128
		var typ uint32
		r, _, _ = procQuery.Call(hKey, uintptr(unsafe.Pointer(pvName)), 0,
			uintptr(unsafe.Pointer(&typ)),
			uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
		procClose.Call(hKey)
		if r == 0 {
			ver := syscall.UTF16ToString(buf[:])
			p("    注册表 %s\n      pv=%s  %s\n", rp, ver, win7VerNote(ver))
			found = true
		}
	}
	if !found {
		p("    注册表: 未检测到系统安装的 WebView2\n")
	}

	// 固定版本目录
	for _, sub := range []string{"webview2"} {
		dir := filepath.Join(baseDir, sub)
		if st, err := os.Stat(filepath.Join(dir, "msedgewebview2.exe")); err == nil && !st.IsDir() {
			p("    固定版本目录 %s\\: msedgewebview2.exe 存在 (%d 字节)\n", sub, st.Size())
			eb := filepath.Join(dir, "EBWebView", "x64", "EmbeddedBrowserWebView.dll")
			if _, err := os.Stat(eb); err == nil {
				p("      EBWebView\\x64\\EmbeddedBrowserWebView.dll: OK（go-webview2 需要）\n")
			} else {
				p("      EBWebView\\x64\\EmbeddedBrowserWebView.dll: 缺失  <-- 运行时目录不完整\n")
			}
		} else {
			p("    固定版本目录 %s\\: 未找到 msedgewebview2.exe\n", sub)
		}
	}
}

func win7VerNote(ver string) string {
	if ver == "" {
		return ""
	}
	major := 0
	fmt.Sscanf(ver, "%d.", &major)
	switch {
	case major >= 110:
		return "(>=110 不支持 Win7，Win7 最高 109.x)"
	case major == 109:
		return "(109.x，Win7 可用)"
	case major >= 94:
		return "(94~108，Win7 可用)"
	default:
		return "(低于 94，版本过旧)"
	}
}

func isAdminUser() bool {
	advapi := syscall.NewLazyDLL("advapi32.dll")
	procAlloc := advapi.NewProc("AllocateAndInitializeSid")
	procCheck := advapi.NewProc("CheckTokenMembership")
	procFree := advapi.NewProc("FreeSid")

	var sid uintptr
	auth := [6]byte{0, 0, 0, 0, 0, 5} // SECURITY_NT_AUTHORITY
	r, _, _ := procAlloc.Call(
		uintptr(unsafe.Pointer(&auth[0])), 2,
		32, 544, 0, 0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&sid)))
	if r == 0 {
		return false
	}
	defer procFree.Call(sid)
	var member int32
	procCheck.Call(0, sid, uintptr(unsafe.Pointer(&member)))
	return member != 0
}

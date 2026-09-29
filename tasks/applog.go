package tasks

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// 应用日志：与 Manager 共用 winvncgo.log，启动阶段即开始记录。
// GUI 程序没有控制台，stderr 上的输出看不到；启动失败必须落到日志文件。
var (
	logMu   sync.Mutex
	logFile *os.File
	logPath string
)

// openAppLog 打开应用日志（exe 目录，不可写则退回 %TEMP%）。
// 重复调用只会打开一次。
func openAppLog(baseDir string) string {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile != nil {
		return logPath
	}

	candidates := []string{filepath.Join(baseDir, "winvncgo.log")}
	if tmp := os.TempDir(); tmp != "" {
		candidates = append(candidates, filepath.Join(tmp, "winvncgo.log"))
	}
	for _, p := range candidates {
		f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			continue
		}
		logFile = f
		logPath = p
		// 把标准库 log 也导向文件，保证 log.Fatal 等不留到看不见的 stderr
		log.SetOutput(io.MultiWriter(f, os.Stderr))
		return logPath
	}
	return ""
}

// appLog 写一行应用日志（带时间戳）。
// 只写日志文件：标准库 log 已被导向同一文件（openAppLog 里 MultiWriter），
// 再 log.Print 会重复记录。
func appLog(msg string) {
	line := fmt.Sprintf("[%s] %s", time.Now().Format("2006-01-02 15:04:05"), msg)
	logMu.Lock()
	f := logFile
	logMu.Unlock()
	if f != nil {
		f.WriteString(line + "\n")
		return
	}
	// 日志文件没打开时退回标准输出，方便开发调试
	log.Print(msg)
}

// appLogf 格式化写一行应用日志
func appLogf(format string, args ...interface{}) {
	appLog(fmt.Sprintf(format, args...))
}

// closeAppLog 关闭日志文件
func closeAppLog() {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

// logStartupEnv 记录启动环境，便于排查 Win7 兼容问题
func logStartupEnv(baseDir string) {
	major, minor, build := osVersion()
	appLogf("========== 启动 ==========")
	appLogf("Go版本: %s  GOOS=%s GOARCH=%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	appLogf("系统版本: Windows %d.%d build %d", major, minor, build)
	appLogf("日志目录: %s", baseDir)
	if exe, err := os.Executable(); err == nil {
		appLogf("可执行文件: %s", exe)
	}
	if wd, err := os.Getwd(); err == nil {
		appLogf("工作目录: %s", wd)
	}
	appLogf("管理员权限: %v", isAdmin())
}

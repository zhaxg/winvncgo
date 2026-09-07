//go:build windows

package tasks

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

const mutexName = "Global\\WinVNCGo_SingleInstance"

var instanceMutex windows.Handle

// acquireInstanceMutex 尝试获取单实例互斥锁。
// 已有实例运行时返回 false（调用方应退出）。
func acquireInstanceMutex() bool {
	h, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(mutexName))
	if err != nil {
		if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
			return false
		}
		// 其他错误（如权限不足），放行
		fmt.Fprintf(os.Stderr, "创建互斥锁失败: %v\n", err)
		return true
	}
	instanceMutex = h
	return true
}

// releaseInstanceMutex 释放互斥锁
func releaseInstanceMutex() {
	if instanceMutex != 0 {
		windows.CloseHandle(instanceMutex)
		instanceMutex = 0
	}
}

// activateExistingWindow 把已有实例的窗口带到前台
func activateExistingWindow() {
	hwnd := findWindowByTitle("远程协助")
	if hwnd == 0 {
		return
	}
	procSetForegroundWindow.Call(hwnd)
}

//go:build windows

package tasks

import (
	"syscall"
	"unsafe"
)

var (
	advapi32                 = syscall.NewLazyDLL("advapi32.dll")
	procCheckTokenMembership = advapi32.NewProc("CheckTokenMembership")
	procAllocateAndInitializeSid = advapi32.NewProc("AllocateAndInitializeSid")
	procFreeSid              = advapi32.NewProc("FreeSid")
)

// sidIdentifierAuthority 对应 Win32 的 SID_IDENTIFIER_AUTHORITY
type sidIdentifierAuthority struct{ Value [6]byte }

// isAdmin 当前进程是否运行在管理员（提权）令牌下。
// winvnc 捕获屏幕需要管理员权限，主进程提权后子进程自动继承；
// 未提权时 VNC 能连上但看不到画面，这里给出明确日志便于排查。
func isAdmin() bool {
	// BUILTIN\Administrators 组 SID: S-1-5-32-544
	const (
		SECURITY_NT_AUTHORITY       = 5
		SECURITY_BUILTIN_DOMAIN_RID = 32
		DOMAIN_ALIAS_RID_ADMINS     = 544
	)
	var adminSid *uintptr
	auth := sidIdentifierAuthority{Value: [6]byte{0, 0, 0, 0, 0, SECURITY_NT_AUTHORITY}}
	r1, _, _ := procAllocateAndInitializeSid.Call(
		uintptr(unsafe.Pointer(&auth)),
		2,
		SECURITY_BUILTIN_DOMAIN_RID,
		DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&adminSid)),
	)
	if r1 == 0 {
		return false
	}
	defer procFreeSid.Call(uintptr(unsafe.Pointer(adminSid)))

	var member int32
	procCheckTokenMembership.Call(
		0,
		uintptr(unsafe.Pointer(adminSid)),
		uintptr(unsafe.Pointer(&member)),
	)
	return member != 0
}

// warnIfNotElevated 未提权时输出告警（winvnc 无法捕获屏幕，连接后黑屏）
func warnIfNotElevated(logger interface{ log(string) }) {
	if isAdmin() {
		return
	}
	logger.log("警告: 程序未以管理员身份运行，winvnc 将无法捕获屏幕（连接后黑屏）。请以管理员身份运行 winvncgo.exe")
}

//go:build windows

package tasks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const serviceName = "uvnc_service"

type VNCService struct {
	mu         sync.Mutex
	isRunning  bool
	hasConn    bool
	serverPath string
	viewerPath string
	configPath string
	port       int
}

func newVNCService(baseDir string) *VNCService {
	return &VNCService{
		serverPath: filepath.Join(baseDir, "winvnc.exe"),
		viewerPath: filepath.Join(baseDir, "vncviewer.exe"),
		configPath: filepath.Join(baseDir, "ultravnc.ini"),
		port:       5900,
	}
}

// openSCM 以完整权限打开服务控制管理器
func openSCM() (windows.Handle, error) {
	return windows.OpenSCManager(nil, nil, windows.SC_MANAGER_ALL_ACCESS)
}

// openServiceRaw 以指定权限打开服务，返回原始 Handle
func openServiceRaw(scm windows.Handle, name string, access uint32) (windows.Handle, error) {
	return windows.OpenService(scm, windows.StringToUTF16Ptr(name), access)
}

// svcFromHandle 将原始 Handle 包装为 *mgr.Service 以使用 Query/Control 方法
func svcFromHandle(h windows.Handle) *mgr.Service {
	return &mgr.Service{Handle: h}
}

// EnsureService 确保 uvnc_service 已安装并运行；不存在则安装，未运行则启动
func (s *VNCService) EnsureService() error {
	if _, err := os.Stat(s.serverPath); os.IsNotExist(err) {
		return fmt.Errorf("winvnc.exe 不存在: %s", s.serverPath)
	}

	scm, err := openSCM()
	if err != nil {
		return fmt.Errorf("连接服务管理器失败: %w", err)
	}
	defer windows.CloseServiceHandle(scm)

	// 尝试打开服务（SERVICE_START 用于启动，SERVICE_QUERY_STATUS 用于查询状态）
	svcH, err := openServiceRaw(scm, serviceName,
		windows.SERVICE_START|windows.SERVICE_QUERY_STATUS)
	if err != nil {
		// 服务不存在，安装并启动
		return s.installAndStartService(scm)
	}
	defer windows.CloseServiceHandle(svcH)

	// 服务存在，检查运行状态
	status, err := svcFromHandle(svcH).Query()
	if err != nil {
		return fmt.Errorf("查询服务状态失败: %w", err)
	}

	if status.State == svc.Running {
		s.mu.Lock()
		s.isRunning = true
		s.mu.Unlock()
		return nil
	}

	// 服务未运行，尝试启动
	if err := svcFromHandle(svcH).Start(); err != nil {
		// 启动失败，删除后重新安装
		windows.CloseServiceHandle(svcH)
		delH, delErr := openServiceRaw(scm, serviceName, windows.DELETE)
		if delErr == nil {
			windows.DeleteService(delH)
			windows.CloseServiceHandle(delH)
			time.Sleep(2 * time.Second)
		}
		return s.installAndStartService(scm)
	}

	time.Sleep(2 * time.Second)
	s.mu.Lock()
	s.isRunning = true
	s.mu.Unlock()
	return nil
}

func (s *VNCService) installAndStartService(scm windows.Handle) error {
	// 确保 ultravnc.portable 存在，让 winvnc 从 exe 同目录读取 ini
	portablePath := filepath.Join(filepath.Dir(s.serverPath), "ultravnc.portable")
	if _, err := os.Stat(portablePath); os.IsNotExist(err) {
		os.WriteFile(portablePath, []byte{}, 0644)
	}

	// 通过 SCM 安装服务，可执行路径为 winvnc.exe -service
	svcPath := `"` + s.serverPath + `" -service`

	svcH, err := windows.CreateService(scm,
		windows.StringToUTF16Ptr(serviceName),       // lpServiceName
		windows.StringToUTF16Ptr("UltraVNC Server"), // lpDisplayName
		windows.SERVICE_ALL_ACCESS,                   // dwDesiredAccess
		windows.SERVICE_WIN32_OWN_PROCESS,            // dwServiceType
		windows.SERVICE_AUTO_START,                   // dwStartType（自动启动）
		windows.SERVICE_ERROR_NORMAL,                 // dwErrorControl
		windows.StringToUTF16Ptr(svcPath),            // lpBinaryPathName
		nil,                                          // lpLoadOrderGroup
		nil,                                          // lpdwTagId
		nil,                                          // lpDependencies
		windows.StringToUTF16Ptr("LocalSystem"),      // lpServiceStartName
		nil,                                          // lpPassword
	)
	if err != nil {
		return fmt.Errorf("安装服务失败: %w", err)
	}
	defer windows.CloseServiceHandle(svcH)

	// 设置延迟自动启动（Automatic Delayed Start）
	// SERVICE_CONFIG_DELAYED_AUTO_START_INFO = 3
	type delayedAutoStartInfo struct {
		fDelayedAutostart int32
	}
	info := delayedAutoStartInfo{fDelayedAutostart: 1}
	windows.ChangeServiceConfig2(svcH, 3, (*byte)(unsafe.Pointer(&info)))

	// 设置 SoftwareSASGeneration 允许服务发送 SAS（Ctrl+Alt+Del）
	key, _, err := registry.CreateKey(
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`,
		registry.SET_VALUE)
	if err == nil {
		key.SetDWordValue("SoftwareSASGeneration", 1)
		key.Close()
	}

	// 直接用创建好的句柄启动（已有 SERVICE_ALL_ACCESS）
	if err := svcFromHandle(svcH).Start(); err != nil {
		return fmt.Errorf("启动服务失败: %w", err)
	}

	time.Sleep(2 * time.Second)
	s.mu.Lock()
	s.isRunning = true
	s.mu.Unlock()
	return nil
}

// SetPassword 将 ID 作为 VNC 密码，用官方算法实时加密后写入配置
func (s *VNCService) SetPassword(id string) error {
	encrypted := vncEncryptPassword(id)

	// 只替换 passwd= 和 passwd2= 等号后面的值，不覆盖其他配置
	s.replacePassword(s.configPath, encrypted)
	return nil
}

// replacePassword 只替换 ini 文件中 passwd= 和 passwd2= 的值，其他内容不动
func (s *VNCService) replacePassword(configPath, encrypted string) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return
	}

	content := string(data)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "passwd=") && !strings.HasPrefix(lower, "passwd2=") {
			lines[i] = "passwd=" + encrypted
		} else if strings.HasPrefix(lower, "passwd2=") {
			lines[i] = "passwd2=" + encrypted
		}
	}
	os.WriteFile(configPath, []byte(strings.Join(lines, "\n")), 0644)
}

func (s *VNCService) writeConfig(configPath, ultravncSection string) {
	existing := ""
	if data, err := os.ReadFile(configPath); err == nil {
		existing = string(data)
	}

	if existing == "" {
		content := ultravncSection + "\n[admin]\nUseRegistry=0\nDisableTrayIcon=0\n"
		os.WriteFile(configPath, []byte(content), 0644)
		return
	}

	updated := replaceSection(existing, "[ultravnc]", ultravncSection)
	os.WriteFile(configPath, []byte(updated), 0644)
}

// Stop 停止 VNC 服务
func (s *VNCService) Stop() {
	scm, err := openSCM()
	if err != nil {
		return
	}
	defer windows.CloseServiceHandle(scm)

	svcH, err := openServiceRaw(scm, serviceName,
		windows.SERVICE_STOP|windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return
	}
	defer windows.CloseServiceHandle(svcH)

	status, err := svcFromHandle(svcH).Query()
	if err != nil {
		return
	}

	if status.State == svc.Running {
		svcFromHandle(svcH).Control(svc.Stop)
	}

	s.mu.Lock()
	s.isRunning = false
	s.mu.Unlock()
}

func (s *VNCService) isServiceRunning() bool {
	scm, err := openSCM()
	if err != nil {
		return false
	}
	defer windows.CloseServiceHandle(scm)

	svcH, err := openServiceRaw(scm, serviceName, windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return false
	}
	defer windows.CloseServiceHandle(svcH)

	status, err := svcFromHandle(svcH).Query()
	if err != nil {
		return false
	}

	return status.State == svc.Running
}

func (s *VNCService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	running := s.isServiceRunning()
	s.isRunning = running
	return running
}

func (s *VNCService) RefreshConnectionState() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.isServiceRunning() {
		s.isRunning = false
		s.hasConn = false
		return
	}
	s.isRunning = true
	s.hasConn = hasEstablishedConnections(s.port)
}

func (s *VNCService) HasActiveConnection() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hasConn
}

func (s *VNCService) LaunchViewer(ip string, port int, password string) error {
	addr := fmt.Sprintf("%s:%d", ip, port)
	args := []string{
		addr,
		"/password", password,
		"/encoding", "ultra2",
		"/notoolbar",
		// "/nohotkeys",
		// "/scale", "2/3",
	}
	return exec.Command(s.viewerPath, args...).Start()
}

// hasEstablishedConnections 通过 netstat 检测是否有已建立的 TCP 连接到指定端口
func hasEstablishedConnections(port int) bool {
	cmd := exec.Command("netstat", "-an")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if !strings.Contains(line, "ESTABLISHED") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		localAddr := parts[1]
		if idx := strings.LastIndex(localAddr, ":"); idx != -1 {
			localPort := localAddr[idx+1:]
			if localPort == fmt.Sprintf("%d", port) {
				return true
			}
		}
	}
	return false
}

// replaceSection 替换 ini 文件中指定段的内容（大小写不敏感）
func replaceSection(content, section, newSection string) string {
	lines := splitLines(content)
	sectionLower := strings.ToLower(section)

	start := -1
	for i, line := range lines {
		if strings.TrimSpace(strings.ToLower(line)) == sectionLower {
			start = i
			break
		}
	}

	if start == -1 {
		return content + "\n" + newSection
	}

	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			end = i
			break
		}
	}

	result := strings.Join(lines[:start], "") + newSection + strings.Join(lines[end:], "")
	return result
}

func splitLines(s string) []string {
	var lines []string
	for len(s) > 0 {
		idx := strings.Index(s, "\n")
		if idx == -1 {
			lines = append(lines, s)
			break
		}
		lines = append(lines, s[:idx+1])
		s = s[idx+1:]
	}
	return lines
}

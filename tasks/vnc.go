package tasks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

type VNCService struct {
	mu         sync.Mutex
	process    *exec.Cmd
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

// SetPassword 将 ID 作为 VNC 密码，用官方算法（vnc_des.go）实时加密后写入配置
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
		// 段不存在，追加到末尾
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

func (s *VNCService) Start() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isRunning {
		return true
	}
	if _, err := os.Stat(s.serverPath); os.IsNotExist(err) {
		return false
	}
	// 确保 ultravnc.portable 存在，让 winvnc 从 exe 同目录读取 ini
	portablePath := filepath.Join(filepath.Dir(s.serverPath), "ultravnc.portable")
	if _, err := os.Stat(portablePath); os.IsNotExist(err) {
		os.WriteFile(portablePath, []byte{}, 0644)
	}
	s.killExisting()
	workDir := filepath.Dir(s.serverPath)
	s.process = exec.Command(s.serverPath, "-run")
	s.process.Dir = workDir
	s.process.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := s.process.Start(); err != nil {
		return false
	}
	s.isRunning = true
	go func() {
		s.process.Wait()
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
	}()
	return true
}

func (s *VNCService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.process != nil && s.process.Process != nil {
		s.process.Process.Kill()
		s.process.Wait()
	}
	s.isRunning = false
}

func (s *VNCService) killExisting() {
	cmd := exec.Command("taskkill", "/F", "/IM", "winvnc.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Run()
}

func (s *VNCService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isRunning
}

func (s *VNCService) RefreshConnectionState() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.process == nil || (s.process.ProcessState != nil && s.process.ProcessState.Exited()) {
		s.isRunning = false
		s.hasConn = false
		return
	}
	s.hasConn = hasEstablishedConnections(s.port)
}

func (s *VNCService) HasActiveConnection() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hasConn
}

func (s *VNCService) LaunchViewer(ip string, port int, password string) error {
	return exec.Command(s.viewerPath, fmt.Sprintf("%s:%d", ip, port), "-password", password).Start()
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
		// 匹配本地端口，如 192.168.1.100:5900 或 [::1]:5900
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

package tasks

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const rotationInterval = 30 * time.Minute

type Manager struct {
	config    Config
	vnc       *VNCService
	rdb       *RedisClient
	localIP   string
	currentID string
	logFile   string
	done      chan struct{}
}

func newManager(cfg Config, baseDir string) *Manager {
	return &Manager{
		config:  cfg,
		vnc:     newVNCService(baseDir),
		logFile: filepath.Join(baseDir, "winvncgo.log"),
		done:    make(chan struct{}),
	}
}

func (m *Manager) Initialize() {
	cc := m.config.ControlCenter()
	if cc.Enabled {
		m.rdb = redisNew(
			cc.Host, cc.Port,
			cc.Password, cc.User,
			cc.KeyPrefix, cc.TTL,
		)
	}
	m.localIP = getLocalIP()
	warnIfNotElevated(m)

	// 确保 VNC 服务已安装并运行
	if err := m.vnc.EnsureService(); err != nil {
		m.log(fmt.Sprintf("VNC 服务启动失败: %v", err))
	}

	m.RefreshID()
	m.log("应用启动成功")
}

// rotationLoop 定时轮换 ID，默认 30 分钟一次
func (m *Manager) rotationLoop() {
	ticker := time.NewTicker(rotationInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := m.RefreshID(); err != nil {
				m.log(fmt.Sprintf("定时轮换跳过: %v", err))
			} else {
				m.log("定时轮换 ID 完成")
			}
		case <-m.done:
			return
		}
	}
}

func (m *Manager) currentStatus() StatusEvent {
	vncStatus := "已停止"
	if m.vnc.IsRunning() {
		vncStatus = "运行中"
	}
	redisStatus := "未连接"
	if m.rdb != nil && m.rdb.IsConnected() {
		redisStatus = "已连接"
	}
	connStatus := "未运行"
	if m.vnc.HasActiveConnection() {
		connStatus = "有活跃连接"
	} else if m.vnc.IsRunning() {
		connStatus = "等待连接"
	}
	return StatusEvent{IP: m.localIP, ID: m.currentID, VNC: vncStatus, Redis: redisStatus, Conn: connStatus}
}

func (m *Manager) GetStatus() map[string]interface{} {
	s := m.currentStatus()
	return map[string]interface{}{
		"localIP":             m.localIP,
		"currentId":           m.currentID,
		"vnc":                 s.VNC,
		"redis":               s.Redis,
		"connection":          s.Conn,
		"hasActiveConnection": m.vnc.HasActiveConnection(),
	}
}

func (m *Manager) RefreshID() error {
	m.vnc.RefreshConnectionState()
	if m.vnc.HasActiveConnection() {
		m.log("当前有活跃连接，跳过 ID 轮换")
		return fmt.Errorf("当前有活跃连接")
	}
	if m.currentID != "" && m.rdb != nil && m.rdb.IsConnected() {
		m.rdb.Del(m.currentID)
	}
	// 不停止 VNC 服务，只更新配置文件中的密码

	newID, err := generateID()
	if err != nil {
		return err
	}
	m.currentID = newID
	m.log(fmt.Sprintf("ID 已生成: %s", newID))

	if err := m.vnc.SetPassword(newID); err != nil {
		m.log(fmt.Sprintf("配置写入失败: %v", err))
		return err
	}
	m.log("密码已更新")

	if m.rdb != nil && m.rdb.IsConnected() {
		port := m.vnc.GetPort()
		if m.rdb.Set(newID, fmt.Sprintf("%s:%d", m.localIP, port)) {
			m.log(fmt.Sprintf("已注册到控制中心 (%s:%d)", m.localIP, port))
		} else {
			m.log("控制中心注册失败")
		}
	}
	return nil
}

func (m *Manager) Connect(remoteID, remoteIP string) error {
	if remoteID == "" {
		return fmt.Errorf("请输入远程 ID")
	}
	targetIP := remoteIP
	targetPort := m.vnc.GetPort()

	if targetIP == "" && m.rdb != nil && m.rdb.IsConnected() {
		m.log("正在查询对方 IP...")
		result, ok := m.rdb.Get(remoteID)
		if ok && result != "" {
			for i := len(result) - 1; i >= 0; i-- {
				if result[i] == ':' {
					targetIP = result[:i]
					fmt.Sscanf(result[i+1:], "%d", &targetPort)
					break
				}
			}
			m.log(fmt.Sprintf("查询成功: %s:%d", targetIP, targetPort))
		} else {
			return fmt.Errorf("未找到该 ID，请手动输入 IP")
		}
	} else if targetIP == "" {
		return fmt.Errorf("控制中心不可用，请输入对方 IP")
	}

	// 手动输入支持 IP:Port 格式，如 192.168.1.100:5901
	if idx := strings.LastIndex(targetIP, ":"); idx != -1 {
		port := 0
		fmt.Sscanf(targetIP[idx+1:], "%d", &port)
		if port > 0 {
			targetIP = targetIP[:idx]
			targetPort = port
		}
	}

	m.log(fmt.Sprintf("正在连接 %s:%d...", targetIP, targetPort))
	return m.vnc.LaunchViewer(targetIP, targetPort, remoteID)
}

func (m *Manager) CopyID() string { return m.currentID }

func (m *Manager) Shutdown() {
	close(m.done)
	// 不停止 VNC 服务，服务保持运行
	if m.currentID != "" && m.rdb != nil && m.rdb.IsConnected() {
		m.rdb.Del(m.currentID)
	}
	if m.rdb != nil {
		m.rdb.Close()
	}
}

func (m *Manager) log(msg string) {
	line := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
	log.Print(line)
	f, err := os.OpenFile(m.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		f.WriteString(line)
		f.Close()
	}
}

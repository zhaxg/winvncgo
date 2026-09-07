package tasks

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Manager struct {
	config    Config
	vnc       *VNCService
	rdb       *RedisClient
	localIP   string
	currentID string
	logFile   string
}

func newManager(cfg Config, baseDir string) *Manager {
	return &Manager{
		config:  cfg,
		vnc:     newVNCService(baseDir),
		logFile: filepath.Join(baseDir, "wegovnc.log"),
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
	m.RefreshID()
	m.log("应用启动成功")
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
		return fmt.Errorf("has active connection")
	}
	if m.currentID != "" && m.rdb != nil && m.rdb.IsConnected() {
		m.rdb.Del(m.currentID)
	}
	m.vnc.Stop()

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
	if !m.vnc.Start() {
		m.log("UltraVNC 启动失败")
		return fmt.Errorf("vnc start failed")
	}
	m.log("UltraVNC 已启动")

	if m.rdb != nil && m.rdb.IsConnected() {
		if m.rdb.Set(newID, fmt.Sprintf("%s:%d", m.localIP, 5900)) {
			m.log("已注册到控制中心")
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
	targetPort := 5900

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

	m.log(fmt.Sprintf("正在连接 %s:%d...", targetIP, targetPort))
	return m.vnc.LaunchViewer(targetIP, targetPort, remoteID)
}

func (m *Manager) CopyID() string { return m.currentID }

func (m *Manager) Shutdown() {
	m.vnc.Stop()
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

package tasks

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config 应用配置，来自 exe 同目录的 winvncgo.json
type Config struct {
	AppSettings AppSettings `json:"AppSettings"`
}

// AppSettings 应用设置
type AppSettings struct {
	// CtlCenter 控制中心连接串，格式：
	//   redis://[user:password@]host[:port][/key-prefix][?ttl=分钟]
	// 示例：
	//   redis://vnc:Vnc2026%@192.168.60.213:6379/vnc?ttl=30
	// 留空表示不启用控制中心
	CtlCenter string `json:"ctl_center"`
}

// ControlCenter 解析后的控制中心连接参数
type ControlCenter struct {
	Enabled   bool
	Host      string
	Port      int
	Password  string
	User      string
	KeyPrefix string // 连接串路径映射为 key 前缀：/vnc -> "vnc:"
	TTL       int    // 注册信息过期时间（分钟）
}

// 默认控制中心参数（连接串未指定时的兜底值）
const (
	defaultHost      = "127.0.0.1"
	defaultPort      = 6379
	defaultKeyPrefix = "vnc:"
	defaultTTL       = 30
)

func loadConfig(path string) Config {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("配置文件未找到，使用默认配置（控制中心禁用）")
		return cfg
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("配置文件解析错误: %v", err)
		return cfg
	}
	log.Printf("配置已加载: %s", path)
	return cfg
}

// ControlCenter 将 ctl_center 连接串解析为连接参数；空串/解析失败返回 Enabled=false
func (c Config) ControlCenter() ControlCenter {
	raw := strings.TrimSpace(c.AppSettings.CtlCenter)
	if raw == "" {
		return ControlCenter{Enabled: false}
	}

	u, err := parseRedisURL(raw)
	if err != nil {
		log.Printf("控制中心连接串无效 (%v)，控制中心禁用", err)
		return ControlCenter{Enabled: false}
	}

	cc := ControlCenter{
		Enabled:   true,
		Host:      u.Hostname(),
		Port:      defaultPort,
		User:      u.User.Username(),
		KeyPrefix: defaultKeyPrefix,
		TTL:       defaultTTL,
	}
	if p, err := strconv.Atoi(u.Port()); err == nil && p > 0 {
		cc.Port = p
	}
	cc.Password, _ = u.User.Password()

	// 路径段映射为 key 前缀：/vnc -> "vnc:"（无路径保持默认前缀）
	if seg := strings.Trim(u.Path, "/"); seg != "" {
		cc.KeyPrefix = seg + ":"
	}

	// 查询参数：ttl=分钟
	if q := u.Query(); q.Get("ttl") != "" {
		if ttl, err := strconv.Atoi(q.Get("ttl")); err == nil && ttl > 0 {
			cc.TTL = ttl
		}
	}
	return cc
}

// parseRedisURL 宽容解析 redis:// 连接串：
// 密码中可能包含未转义的特殊字符（如 %），先补全转义再交给 url.Parse
func parseRedisURL(raw string) (*url.URL, error) {
	if !strings.Contains(raw, "://") {
		raw = "redis://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		// 常见失败原因：userinfo 中有 % 等未转义字符，转义后重试
		if fixed, fixErr := escapeUserInfo(raw); fixErr == nil {
			if u2, parseErr := url.Parse(fixed); parseErr == nil {
				return u2, nil
			}
		}
		return nil, err
	}
	return u, nil
}

// escapeUserInfo 转义 userinfo 中密码部分的非法字符
func escapeUserInfo(raw string) (string, error) {
	scheme := ""
	rest := raw
	if i := strings.Index(raw, "://"); i != -1 {
		scheme, rest = raw[:i+3], raw[i+3:]
	}
	at := strings.LastIndex(rest, "@")
	if at == -1 {
		return "", fmt.Errorf("连接串中缺少用户名密码")
	}
	userinfo := rest[:at]
	host := rest[at+1:]

	colon := strings.Index(userinfo, ":")
	var user, pass string
	if colon == -1 {
		user = userinfo
	} else {
		user, pass = userinfo[:colon], userinfo[colon+1:]
	}
	return scheme + url.PathEscape(user) + ":" + url.PathEscape(pass) + "@" + host, nil
}

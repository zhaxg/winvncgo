package tasks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadConfigSample 对齐 libs/wegovnc.json 示例配置的解析结果
func TestLoadConfigSample(t *testing.T) {
	const sample = `{
  "AppSettings": {
    "ctl_center": "redis://vnc:Vnc2026%@192.168.60.213:6379/vnc?ttl=30"
  }
}`
	path := filepath.Join(t.TempDir(), "wegovnc.json")
	if err := os.WriteFile(path, []byte(sample), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := loadConfig(path)
	cc := cfg.ControlCenter()
	if !cc.Enabled {
		t.Fatal("控制中心应启用")
	}
	if cc.Host != "192.168.60.213" {
		t.Errorf("Host = %q, want %q", cc.Host, "192.168.60.213")
	}
	if cc.Port != 6379 {
		t.Errorf("Port = %d, want 6379", cc.Port)
	}
	if cc.User != "vnc" {
		t.Errorf("User = %q, want %q", cc.User, "vnc")
	}
	if cc.Password != "Vnc2026%" {
		t.Errorf("Password = %q, want %q（密码含未转义的 %% 应宽容解析）", cc.Password, "Vnc2026%")
	}
	if cc.KeyPrefix != "vnc:" {
		t.Errorf("KeyPrefix = %q, want %q（路径 /vnc 应映射为前缀 vnc:，与旧格式 key 兼容）", cc.KeyPrefix, "vnc:")
	}
	if cc.TTL != 30 {
		t.Errorf("TTL = %d, want 30", cc.TTL)
	}
}

// TestControlCenterDefaults 连接串缺省字段时使用默认值
func TestControlCenterDefaults(t *testing.T) {
	cfg := Config{AppSettings: AppSettings{CtlCenter: "redis://192.168.1.10"}}
	cc := cfg.ControlCenter()
	if !cc.Enabled {
		t.Fatal("控制中心应启用")
	}
	if cc.Host != "192.168.1.10" {
		t.Errorf("Host = %q, want %q", cc.Host, "192.168.1.10")
	}
	if cc.Port != 6379 {
		t.Errorf("Port = %d, want 默认 6379", cc.Port)
	}
	if cc.KeyPrefix != "vnc:" {
		t.Errorf("KeyPrefix = %q, want 默认 %q", cc.KeyPrefix, "vnc:")
	}
	if cc.TTL != 30 {
		t.Errorf("TTL = %d, want 默认 30", cc.TTL)
	}
	if cc.User != "" || cc.Password != "" {
		t.Errorf("User/Password = %q/%q, want 空", cc.User, cc.Password)
	}
}

// TestControlCenterDisabled 空 ctl_center 表示不启用控制中心
func TestControlCenterDisabled(t *testing.T) {
	for _, raw := range []string{"", "   "} {
		cfg := Config{AppSettings: AppSettings{CtlCenter: raw}}
		if cc := cfg.ControlCenter(); cc.Enabled {
			t.Errorf("ctl_center = %q 时控制中心应禁用", raw)
		}
	}
	// 默认零值配置也应禁用
	zero := Config{}
	if cc := zero.ControlCenter(); cc.Enabled {
		t.Error("空配置时控制中心应禁用")
	}
}

// TestControlCenterInvalidURL 无效连接串应禁用而不是 panic
func TestControlCenterInvalidURL(t *testing.T) {
	cfg := Config{AppSettings: AppSettings{CtlCenter: "redis://[bad"}}
	if cc := cfg.ControlCenter(); cc.Enabled {
		t.Error("无效连接串时控制中心应禁用")
	}
}

// TestControlCenterVariants 各种连接串变体
func TestControlCenterVariants(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		host      string
		port      int
		user      string
		password  string
		keyPrefix string
		ttl       int
	}{
		{
			"带端口和前缀",
			"redis://:secret@10.0.0.5:6380/myprefix?ttl=5",
			"10.0.0.5", 6380, "", "secret", "myprefix:", 5,
		},
		{
			"仅用户无密码",
			"redis://default@127.0.0.1:6379/vnc",
			"127.0.0.1", 6379, "default", "", "vnc:", 30,
		},
		{
			"密码含特殊字符",
			"redis://vnc:p@ssw0rd!@10.0.0.1:6379/vnc",
			"10.0.0.1", 6379, "vnc", "p@ssw0rd!", "vnc:", 30,
		},
		{
			"无协议前缀",
			"192.168.60.213:6379",
			"192.168.60.213", 6379, "", "", "vnc:", 30,
		},
		{
			"多级路径前缀",
			"redis://10.0.0.1:6379/a/b?ttl=1",
			"10.0.0.1", 6379, "", "", "a/b:", 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{AppSettings: AppSettings{CtlCenter: tt.raw}}
			cc := cfg.ControlCenter()
			if !cc.Enabled {
				t.Fatalf("控制中心应启用")
			}
			if cc.Host != tt.host {
				t.Errorf("Host = %q, want %q", cc.Host, tt.host)
			}
			if cc.Port != tt.port {
				t.Errorf("Port = %d, want %d", cc.Port, tt.port)
			}
			if cc.User != tt.user {
				t.Errorf("User = %q, want %q", cc.User, tt.user)
			}
			if cc.Password != tt.password {
				t.Errorf("Password = %q, want %q", cc.Password, tt.password)
			}
			if cc.KeyPrefix != tt.keyPrefix {
				t.Errorf("KeyPrefix = %q, want %q", cc.KeyPrefix, tt.keyPrefix)
			}
			if cc.TTL != tt.ttl {
				t.Errorf("TTL = %d, want %d", cc.TTL, tt.ttl)
			}
		})
	}
}

// TestConfigJSONTag 确认 JSON 字段名与配置文件格式一致
func TestConfigJSONTag(t *testing.T) {
	var cfg Config
	if err := json.Unmarshal([]byte(`{"AppSettings":{"ctl_center":"redis://h:1/x"}}`), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.AppSettings.CtlCenter != "redis://h:1/x" {
		t.Errorf("ctl_center 未正确解析: %q", cfg.AppSettings.CtlCenter)
	}
}

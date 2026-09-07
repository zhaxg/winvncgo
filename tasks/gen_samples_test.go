package tasks

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// officialVncpwd 用官方 vncpwd.exe 解密16位hex密文，返回明文
func officialVncpwd(t *testing.T, hex16 string) string {
	t.Helper()
	vncpwd := `..\libs\vncpwd\vncpwd.exe`
	if _, err := os.Stat(vncpwd); os.IsNotExist(err) {
		t.Skip("vncpwd.exe not found, skipping cross-validation")
	}
	cmd := exec.Command(vncpwd, hex16)
	cmd.Stdin = strings.NewReader("\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("vncpwd.exe failed: %v\n%s", err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "Password:") {
			continue
		}
		// 先去行首空格再剥前缀，最后清理冒号后的空白
		trimmed := strings.TrimSpace(line)
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "Password:"))
	}
	t.Fatalf("vncpwd.exe output missing Password: line:\n%s", out)
	return ""
}

// TestVncEncryptMatchesOfficial 用官方 vncpwd.exe 验证 Go 加密结果
func TestVncEncryptMatchesOfficial(t *testing.T) {
	passwords := []string{
		// 基础长度
		"1", "12", "123", "1234", "12345", "123456", "1234567", "12345678",
		// 10 个预生成 ID
		"114514", "823456", "773890", "226100", "551789",
		"330264", "998271", "664038", "109675", "447231",
		// 纯字母
		"a", "ab", "abc", "abcdef", "ABCDEF", "abcdefgh", "ABCDEFGH",
		// 混合字符
		"a1b2c3", "z9z9z9z9", "test1234", "p@ssw0rd",
		// 边界值
		"000000", "00000000", "999999", "ffffffff",
		// 大小写混合
		"AbCdEfGh", "1a2B3c4D",
		// 特殊字符
		"Hello!!", "a b c", "pass word",
		// 截断测试（官方 MAXPWLEN=8：>8 字符只取前 8，明文 9 字符预期解密为前 8 字符）
		"123456789", "longpassword",
		// 更多组合
		"01234567", "87654321", "xX1234", "test!!!",
	}

	for _, pw := range passwords {
		// 官方行为：密码最长 8 字符，超出部分截断
		expected := pw
		if len(expected) > 8 {
			expected = expected[:8]
		}
		t.Run(pw, func(t *testing.T) {
			encrypted := vncEncryptPassword(pw)
			if len(encrypted) != 18 {
				t.Fatalf("vncEncryptPassword(%q) returned %d hex chars, want 18 (16 位密文 + 2 位校验和)", pw, len(encrypted))
			}

			// 用官方 vncpwd.exe 解密 Go 加密的密文（剥离末尾 2 位校验和）
			decrypted := officialVncpwd(t, encrypted[:16])
			if decrypted != expected {
				t.Errorf("vncpwd.exe(Go encrypt(%q)) = %q, want %q\n  Go encrypted: %s", pw, decrypted, expected, encrypted)
			}

			// 同时验证 Go 自己的 roundtrip
			roundtrip := vncDecryptHex(encrypted)
			if roundtrip != expected {
				t.Errorf("vncDecryptHex(Go encrypt(%q)) = %q, want %q", pw, roundtrip, expected)
			}
		})
	}
}

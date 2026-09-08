package tasks

import (
	"encoding/hex"
	"fmt"
	"testing"
)

// 本文件维护 preGeneratedPasswords 对照表（原存于 tasks/vnc.go，现移入测试），
// 并验证 tasks/vnc_des.go 的加解密实现与官方算法完全一致，算法对齐官方原版：
//   - 加密：libs/setpasswd/setpasswd.cpp 的 vncEncryptPasswd —— 密码补零到
//     8 字节，用固定密钥 {23,82,107,6,35,78,88,7} 做 DES(EN0) 加密
//   - 存储：libs/setpasswd/inifile.cpp 经 WritePrivateProfileStruct 写入 ini，
//     格式为 8 字节密文 hex(16 位) + 1 字节校验和 hex(2 位)，共 18 位
//   - 解密：libs/vncpwd/vncpwd.c —— hex 解码取前 8 字节做 DES(DE1) 解密
//
// 生产代码不再查表：vnc.go 的 SetPassword 按 ID 实时计算加密密码后写入配置，
// 本表仅作为官方格式的已知答案回归用例
var preGeneratedPasswords = map[string]string{
	"114514": "4821E35B83CBA3059D",
	"823456": "A102B841EE49EEBA7B",
	"773890": "EA3A97DA7D7BD160BE",
	"226100": "1EC949A8ACECC30134",
	"551789": "4E97A52DA9C0A20CCE",
	"330264": "CCE0377AE4754BEFF0",
	"998271": "2A6BD30C74B57F96B2",
	"664038": "9C524291C09AF30210",
	"109675": "5BF0F51C5A669E31EB",
	"447231": "A6A4566F4DA076B729",
}

// TestPreGeneratedPasswordsEncrypt 对照表逐条验证：Go 加密结果必须与对照表完全一致
func TestPreGeneratedPasswordsEncrypt(t *testing.T) {
	for id, stored := range preGeneratedPasswords {
		t.Run(id, func(t *testing.T) {
			if got := vncEncryptPassword(id); got != stored {
				t.Errorf("vncEncryptPassword(%q) = %q, want %q", id, got, stored)
			}
		})
	}
}

// TestPreGeneratedPasswordsDecrypt 对照表逐条验证：解密对照表中的加密字符串必须还原出 ID
func TestPreGeneratedPasswordsDecrypt(t *testing.T) {
	for id, stored := range preGeneratedPasswords {
		t.Run(id, func(t *testing.T) {
			if got := vncDecryptHex(stored); got != id {
				t.Errorf("vncDecryptHex(%q) = %q, want %q", stored, got, id)
			}
		})
	}
}

// TestPreGeneratedPasswordsChecksum 验证对照表存储格式：
// 末 2 位校验和 = 前 8 字节密文之和 mod 256（WritePrivateProfileStruct 追加）
func TestPreGeneratedPasswordsChecksum(t *testing.T) {
	for id, stored := range preGeneratedPasswords {
		t.Run(id, func(t *testing.T) {
			if len(stored) != 18 {
				t.Fatalf("对照表 %q 长度 = %d, want 18 (16 位密文 + 2 位校验和)", stored, len(stored))
			}
			data, err := hex.DecodeString(stored[:16])
			if err != nil {
				t.Fatalf("对照表 %q 不是合法 hex: %v", stored, err)
			}
			sum := 0
			for _, b := range data {
				sum += int(b)
			}
			if got := fmt.Sprintf("%02X", sum%256); got != stored[16:] {
				t.Errorf("对照表 %q 校验和错误: got %s, want %s", stored, got, stored[16:])
			}
		})
	}
}

// TestGenerateID 验证新 ID 生成逻辑
func TestGenerateID(t *testing.T) {
	allowedChars := "0123456789AEKPRSTUVWX"
	for i := 0; i < 100; i++ {
		id, err := generateID()
		if err != nil {
			t.Fatalf("generateID() 返回错误: %v", err)
		}
		if len(id) != 6 {
			t.Errorf("generateID() 返回长度 %d, 期望 6", len(id))
		}
		for _, c := range id {
			found := false
			for _, ac := range allowedChars {
				if c == ac {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("generateID() 返回包含非法字符 %c", c)
			}
		}
	}
}

// TestVncEncryptPasswordKnownVector 已知答案：libs/ultravnc/ultravnc.ini 中的
// passwd=494015F9A35E8B2245 即明文 123456 的官方存储结果
func TestVncEncryptPasswordKnownVector(t *testing.T) {
	const (
		plaintext = "123456"
		stored    = "494015F9A35E8B2245"
	)
	if got := vncEncryptPassword(plaintext); got != stored {
		t.Errorf("vncEncryptPassword(%q) = %q, want %q", plaintext, got, stored)
	}
	if got := vncDecryptHex(stored); got != plaintext {
		t.Errorf("vncDecryptHex(%q) = %q, want %q", stored, got, plaintext)
	}
}

// TestVncEncryptDecryptRoundtrip 任意密码加解密 roundtrip
func TestVncEncryptDecryptRoundtrip(t *testing.T) {
	passwords := []string{
		"1", "12", "123", "1234", "12345", "123456", "1234567", "12345678",
		"abcdef", "ABCDEFG", "TestPass", "a1b2c3", "p@ssw0rd",
	}
	for _, pw := range passwords {
		t.Run(pw, func(t *testing.T) {
			encrypted := vncEncryptPassword(pw)
			if len(encrypted) != 18 {
				t.Errorf("vncEncryptPassword(%q) 长度 = %d, want 18", pw, len(encrypted))
			}
			if decrypted := vncDecryptHex(encrypted); decrypted != pw {
				t.Errorf("roundtrip 失败: encrypt(%q)=%q, decrypt=%q", pw, encrypted, decrypted)
			}
		})
	}
}

// TestVncDecryptHexFormats 解密同时兼容两种 hex 格式：
// 18 位（16 位密文 + 2 位校验和，官方 ini 存储格式）与 16 位纯密文
func TestVncDecryptHexFormats(t *testing.T) {
	const (
		plaintext = "114514"
		stored    = "4821E35B83CBA3059D"
	)
	if got := vncDecryptHex(stored); got != plaintext {
		t.Errorf("18 位格式解密失败: vncDecryptHex(%q) = %q, want %q", stored, got, plaintext)
	}
	if got := vncDecryptHex(stored[:16]); got != plaintext {
		t.Errorf("16 位格式解密失败: vncDecryptHex(%q) = %q, want %q", stored[:16], got, plaintext)
	}
}

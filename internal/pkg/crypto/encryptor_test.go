package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// TestNewEncryptor 測試密鑰加載與初始化
func TestNewEncryptor(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test.key")

	// 1. 測試：文件不存在時自動生成
	// 注意：取決於您的實現，如果 NewEncryptor 不會自動生成，這裡可能會失敗。
	// 根據常見做法，如果我們預期它讀取现有文件，我們先創建一個。

	rawKey := make([]byte, 32)
	rand.Read(rawKey)
	keyHex := hex.EncodeToString(rawKey)

	if err := os.WriteFile(keyPath, []byte(keyHex), 0600); err != nil {
		t.Fatal(err)
	}

	enc, err := NewEncryptor(keyPath)
	if err != nil {
		t.Fatalf("Failed to create encryptor with valid key: %v", err)
	}
	if enc == nil {
		t.Fatal("Encryptor instance is nil")
	}

	// 2. 測試：無效密鑰 (長度錯誤)
	badKeyPath := filepath.Join(tempDir, "bad.key")
	os.WriteFile(badKeyPath, []byte("short-key"), 0600)

	_, err = NewEncryptor(badKeyPath)
	if err == nil {
		t.Error("NewEncryptor should fail with invalid key length")
	}
}

// TestEncryptDecrypt 測試加密解密完整流程
func TestEncryptDecrypt(t *testing.T) {
	// 準備環境
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "enc.key")

	// 寫入合法密鑰
	rawKey := make([]byte, 32)
	rand.Read(rawKey)
	os.WriteFile(keyPath, []byte(hex.EncodeToString(rawKey)), 0600)

	enc, _ := NewEncryptor(keyPath)

	plainText := "Hello, Prism V2!"

	// 1. 加密
	cipherText, err := enc.Encrypt(plainText)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if cipherText == plainText {
		t.Error("Ciphertext should not match plaintext")
	}
	// 簡單驗證是否為 Hex 或 Base64 (取決於您的實現，通常是 Hex 或 Base64 字符串)
	if len(cipherText) == 0 {
		t.Error("Ciphertext is empty")
	}

	// 2. 解密
	decrypted, err := enc.Decrypt(cipherText)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plainText {
		t.Errorf("Decryption mismatch.\nWant: %s\nGot: %s", plainText, decrypted)
	}
}

// TestHMAC 測試完整性校驗
func TestHMAC(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "hmac.key")

	rawKey := make([]byte, 32)
	rand.Read(rawKey)
	os.WriteFile(keyPath, []byte(hex.EncodeToString(rawKey)), 0600)

	enc, _ := NewEncryptor(keyPath)

	data := []byte("Important Config Data")

	// 1. 計算 HMAC
	signature := enc.ComputeHMAC(data)
	if signature == "" {
		t.Fatal("HMAC signature is empty")
	}

	// 2. 驗證正確的簽名
	if !enc.VerifyHMAC(data, signature) {
		t.Error("HMAC verification failed for valid data")
	}

	// 3. 驗證被篡改的數據
	tamperedData := []byte("Important Config Data (Hacked)")
	if enc.VerifyHMAC(tamperedData, signature) {
		t.Error("HMAC verification should fail for tampered data")
	}

	// 4. 驗證錯誤的簽名
	if enc.VerifyHMAC(data, "invalid-signature") {
		t.Error("HMAC verification should fail for invalid signature")
	}
}

// TestIsEncrypted 測試判斷字符串是否已加密的輔助函數
func TestIsEncrypted(t *testing.T) {
	// 假設您的判斷邏輯是檢查是否有特定前綴，或者是否為 IV+Cipher 格式
	// 這裡測試一些常見情況

	tests := []struct {
		input    string
		expected bool
	}{
		{"Plain text", false},
		{"", false},
		// 這裡假設您的 IsEncrypted 邏輯比較寬鬆或依賴特定特徵
		// 如果無法確定，這個測試可能需要根據您的代碼調整
	}

	for _, tt := range tests {
		// 這裡調用包級函數 IsEncrypted
		if got := IsEncrypted(tt.input); got != tt.expected {
			t.Errorf("IsEncrypted(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	// EncryptedPrefix 加密值的前綴標識
	EncryptedPrefix = "enc:"
	// KeySize AES-256 密鑰長度
	KeySize = 32
)

// Encryptor 加密器 (單密鑰版)
type Encryptor struct {
	key []byte
}

// NewEncryptor 創建加密器
// 接收 keyPath string，與 wire.go 中的調用匹配
func NewEncryptor(keyPath string) (*Encryptor, error) {
	// 1. 優先從環境變量讀取
	if keyHex := os.Getenv("PRISM_MASTER_KEY"); keyHex != "" {
		key, err := decodeKey(keyHex)
		if err == nil {
			return &Encryptor{key: key}, nil
		}
		return nil, fmt.Errorf("環境變量 PRISM_MASTER_KEY 格式錯誤: %w", err)
	}

	// 2. 嘗試從文件讀取
	if _, err := os.Stat(keyPath); err == nil {
		content, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("無法讀取密鑰文件: %w", err)
		}
		// 處理可能的空白字符
		keyStr := strings.TrimSpace(string(content))
		key, err := decodeKey(keyStr)
		if err != nil {
			return nil, fmt.Errorf("密鑰文件內容無效: %w", err)
		}
		return &Encryptor{key: key}, nil
	}

	// 3. 自動生成並保存 (首次運行)
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("生成隨機密鑰失敗: %w", err)
	}

	if err := atomicWriteKey(keyPath, key); err != nil {
		return nil, fmt.Errorf("保存新密鑰失敗: %w", err)
	}

	return &Encryptor{key: key}, nil
}

// atomicWriteKey 原子寫入密鑰文件
func atomicWriteKey(filename string, key []byte) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	// 使用臨時文件確保寫入原子性
	tmpFile, err := os.CreateTemp(dir, ".masterkey.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name()) // 清理臨時文件

	// 保存為 Hex 字符串
	if _, err := tmpFile.WriteString(hex.EncodeToString(key)); err != nil {
		tmpFile.Close()
		return err
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	// 設置權限 600
	if err := os.Chmod(tmpFile.Name(), 0600); err != nil {
		return err
	}

	return os.Rename(tmpFile.Name(), filename)
}

func decodeKey(input string) ([]byte, error) {
	// 嘗試 Hex
	key, err := hex.DecodeString(input)
	if err == nil && len(key) == KeySize {
		return key, nil
	}
	// 嘗試 Base64 (兼容舊格式)
	key, err = base64.StdEncoding.DecodeString(input)
	if err == nil && len(key) == KeySize {
		return key, nil
	}
	return nil, errors.New("無效的密鑰格式或長度")
}

// Encrypt 加密
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return EncryptedPrefix + base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密
func (e *Encryptor) Decrypt(encrypted string) (string, error) {
	if !IsEncrypted(encrypted) {
		return "", errors.New("數據未加密")
	}

	raw := strings.TrimPrefix(encrypted, EncryptedPrefix)
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("密文數據過短")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("解密失敗: %w", err)
	}

	return string(plaintext), nil
}

// IsEncrypted 檢查字符串是否已加密
func IsEncrypted(text string) bool {
	return strings.HasPrefix(text, EncryptedPrefix)
}

// ComputeHMAC 計算 HMAC (用於備份完整性)
// 接收 []byte, 返回 string hex
func (e *Encryptor) ComputeHMAC(data []byte) string {
	h := hmac.New(sha256.New, e.key)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyHMAC 驗證 HMAC (防止計時攻擊)
func (e *Encryptor) VerifyHMAC(data []byte, expectedHex string) bool {
	actualHex := e.ComputeHMAC(data)
	return subtle.ConstantTimeCompare([]byte(actualHex), []byte(expectedHex)) == 1
}

// GetKeyInfo 獲取密鑰信息 (用於調試)
func (e *Encryptor) GetKeyInfo() string {
	if len(e.key) < 4 {
		return "invalid"
	}
	return fmt.Sprintf("AES-256 (prefix: %x...)", e.key[:4])
}

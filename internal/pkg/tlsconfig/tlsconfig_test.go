package tlsconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCertChain_NonExistent(t *testing.T) {
	err := ValidateCertChain("/nonexistent/cert.pem", "/nonexistent/key.pem")
	assert.Error(t, err, "不存在的文件应该返回错误")
}

func TestValidateCertChain_EmptyFiles(t *testing.T) {
	tmpDir := t.TempDir()

	certPath := filepath.Join(tmpDir, "empty.crt")
	keyPath := filepath.Join(tmpDir, "empty.key")

	// 创建空文件
	err := os.WriteFile(certPath, []byte{}, 0644)
	assert.NoError(t, err)

	err = os.WriteFile(keyPath, []byte{}, 0600)
	assert.NoError(t, err)

	// 空文件应该验证失败
	err = ValidateCertChain(certPath, keyPath)
	assert.Error(t, err, "空文件应该验证失败")
}

func TestValidateCertChain_InvalidContent(t *testing.T) {
	tmpDir := t.TempDir()

	certPath := filepath.Join(tmpDir, "invalid.crt")
	keyPath := filepath.Join(tmpDir, "invalid.key")

	// 创建无效内容的文件
	err := os.WriteFile(certPath, []byte("not a valid certificate"), 0644)
	assert.NoError(t, err)

	err = os.WriteFile(keyPath, []byte("not a valid key"), 0600)
	assert.NoError(t, err)

	// 无效内容应该验证失败
	err = ValidateCertChain(certPath, keyPath)
	assert.Error(t, err, "无效的证书内容应该验证失败")
}

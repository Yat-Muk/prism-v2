package certinfo

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRepository(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewRepository(tmpDir)

	assert.NotNil(t, repo)
}

func TestRepository_ListCerts_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewRepository(tmpDir)

	certs, err := repo.ListCerts()
	require.NoError(t, err)
	assert.Empty(t, certs)
}

func TestRepository_ListCerts_NonExistentDir(t *testing.T) {
	nonExistentDir := filepath.Join(t.TempDir(), "nonexistent")
	repo := NewRepository(nonExistentDir)

	// 防御性编程：目录不存在应返回空列表而非错误
	certs, err := repo.ListCerts()
	require.NoError(t, err)
	assert.Empty(t, certs)
}

func TestRepository_ListCerts_WithACMECerts(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewRepository(tmpDir)

	// 创建测试证书文件
	domain := "test.example.com"
	certPath := filepath.Join(tmpDir, domain+".crt")

	// 使用辅助函数生成测试证书
	certPEM, _ := generateTestCert(domain, time.Now().Add(365*24*time.Hour))
	err := os.WriteFile(certPath, certPEM, 0644)
	require.NoError(t, err)

	// 获取证书列表
	certs, err := repo.ListCerts()
	require.NoError(t, err)

	// 验证
	assert.Len(t, certs, 1)
	assert.Equal(t, domain, certs[0].Domain)
}

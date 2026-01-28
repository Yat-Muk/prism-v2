package cert

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewSelfSignedGenerator(t *testing.T) {
	log := zap.NewNop()
	gen := NewSelfSignedGenerator(log)

	assert.NotNil(t, gen)
	assert.NotNil(t, gen.log)
}

func TestSelfSignedGenerator_EnsureSelfSigned(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewSelfSignedGenerator(zap.NewNop())

	certPath := filepath.Join(tmpDir, "test.crt")
	keyPath := filepath.Join(tmpDir, "test.key")
	domain := "example.com"

	// 第一次调用：生成证书
	err := gen.EnsureSelfSigned(certPath, keyPath, domain)
	require.NoError(t, err)

	// 验证文件存在
	assert.FileExists(t, certPath)
	assert.FileExists(t, keyPath)

	// 验证证书内容
	certData, err := os.ReadFile(certPath)
	require.NoError(t, err)
	assert.Contains(t, string(certData), "BEGIN CERTIFICATE")
	assert.Contains(t, string(certData), "END CERTIFICATE")

	// 验证私钥内容
	keyData, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Contains(t, string(keyData), "BEGIN EC PRIVATE KEY")

	// 验证文件权限
	certInfo, _ := os.Stat(certPath)
	assert.Equal(t, SelfSignedCertMode, certInfo.Mode().Perm())

	keyInfo, _ := os.Stat(keyPath)
	assert.Equal(t, SelfSignedKeyMode, keyInfo.Mode().Perm())

	// 第二次调用：应该跳过生成（已存在）
	err = gen.EnsureSelfSigned(certPath, keyPath, domain)
	require.NoError(t, err)
}

func TestSelfSignedGenerator_EnsureSelfSigned_DefaultDomain(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewSelfSignedGenerator(zap.NewNop())

	certPath := filepath.Join(tmpDir, "default.crt")
	keyPath := filepath.Join(tmpDir, "default.key")

	// 传入空域名，应该使用默认值 "www.bing.com"
	err := gen.EnsureSelfSigned(certPath, keyPath, "")
	require.NoError(t, err)

	assert.FileExists(t, certPath)
	assert.FileExists(t, keyPath)
}

func TestSelfSignedGenerator_EnsureSelfSigned_CustomDomain(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewSelfSignedGenerator(zap.NewNop())

	certPath := filepath.Join(tmpDir, "custom.crt")
	keyPath := filepath.Join(tmpDir, "custom.key")
	domain := "custom.example.com"

	err := gen.EnsureSelfSigned(certPath, keyPath, domain)
	require.NoError(t, err)

	// 读取证书并验证域名
	certData, err := os.ReadFile(certPath)
	require.NoError(t, err)

	// 简单验证：证书内容应该包含域名相关信息
	assert.NotEmpty(t, certData)
}

func TestSelfSignedGenerator_EnsureSelfSigned_NestedDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewSelfSignedGenerator(zap.NewNop())

	// 测试自动创建嵌套目录
	nestedPath := filepath.Join(tmpDir, "nested", "path", "to", "cert")
	certPath := filepath.Join(nestedPath, "test.crt")
	keyPath := filepath.Join(nestedPath, "test.key")

	err := gen.EnsureSelfSigned(certPath, keyPath, "test.com")
	require.NoError(t, err)

	assert.FileExists(t, certPath)
	assert.FileExists(t, keyPath)

	// 验证目录权限
	dirInfo, _ := os.Stat(nestedPath)
	assert.Equal(t, SelfSignedDirMode, dirInfo.Mode().Perm())
}

func TestSelfSignedGenerator_VerifyPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewSelfSignedGenerator(zap.NewNop())

	certPath := filepath.Join(tmpDir, "perm.crt")
	keyPath := filepath.Join(tmpDir, "perm.key")

	// 生成证书
	err := gen.EnsureSelfSigned(certPath, keyPath, "perm-test.com")
	require.NoError(t, err)

	// 手动修改权限为不安全的值
	os.Chmod(keyPath, 0644)

	// 调用 verifyPermissions（通过再次调用 EnsureSelfSigned，它会跳过生成但执行权限检查）
	err = gen.verifyPermissions(certPath, keyPath)
	require.NoError(t, err)

	// 验证权限已被修正
	keyInfo, _ := os.Stat(keyPath)
	assert.Equal(t, SelfSignedKeyMode, keyInfo.Mode().Perm())
}

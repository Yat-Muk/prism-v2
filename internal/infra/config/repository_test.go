package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewFileRepository(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	repo := NewFileRepository(configPath, nil, zap.NewNop())

	assert.NotNil(t, repo)
	assert.Equal(t, configPath, repo.filePath)
}

func TestFileRepository_Load_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.yaml")

	repo := NewFileRepository(configPath, nil, zap.NewNop())
	ctx := context.Background()

	// 文件不存在应返回默认配置
	cfg, err := repo.Load(ctx)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestFileRepository_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	repo := NewFileRepository(configPath, nil, zap.NewNop())
	ctx := context.Background()

	// 创建测试配置
	cfg := domainConfig.DefaultConfig()

	// 修改一些可验证的字段
	cfg.Certificate.DNSProvider = "cloudflare"
	cfg.Protocols.Hysteria2.Enabled = true
	cfg.Protocols.Hysteria2.Port = 8443

	// 保存
	err := repo.Save(ctx, cfg)
	require.NoError(t, err)

	// 验证文件存在
	assert.FileExists(t, configPath)

	// 验证文件权限
	info, _ := os.Stat(configPath)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// 加载
	loadedCfg, err := repo.Load(ctx)
	require.NoError(t, err)

	// 验证数据一致性
	assert.Equal(t, "cloudflare", loadedCfg.Certificate.DNSProvider)
	assert.True(t, loadedCfg.Protocols.Hysteria2.Enabled)
	assert.Equal(t, 8443, loadedCfg.Protocols.Hysteria2.Port)
}

func TestFileRepository_Cache(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	repo := NewFileRepository(configPath, nil, zap.NewNop())
	ctx := context.Background()

	// 保存配置
	cfg := domainConfig.DefaultConfig()
	cfg.Certificate.DNSProvider = "route53"
	err := repo.Save(ctx, cfg)
	require.NoError(t, err)

	// 第一次加载（从文件）
	cfg1, err := repo.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, "route53", cfg1.Certificate.DNSProvider)

	// 第二次加载（从缓存）
	cfg2, err := repo.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, "route53", cfg2.Certificate.DNSProvider)

	// 验证返回的是不同的实例（深拷贝）
	cfg2.Certificate.DNSProvider = "modified"
	assert.Equal(t, "route53", cfg1.Certificate.DNSProvider) // cfg1不应受影响
}

func TestFileRepository_HotReload(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	repo := NewFileRepository(configPath, nil, zap.NewNop())
	ctx := context.Background()

	// 初始保存
	cfg1 := domainConfig.DefaultConfig()
	cfg1.Protocols.Hysteria2.Port = 9001
	repo.Save(ctx, cfg1)

	// 加载
	loaded1, _ := repo.Load(ctx)
	assert.Equal(t, 9001, loaded1.Protocols.Hysteria2.Port)

	// 等待一小段时间确保文件修改时间变化
	time.Sleep(10 * time.Millisecond)

	// 外部修改文件
	cfg2 := domainConfig.DefaultConfig()
	cfg2.Protocols.Hysteria2.Port = 9002
	repo.Save(ctx, cfg2)

	// 再次加载应该检测到变化
	loaded2, _ := repo.Load(ctx)
	assert.Equal(t, 9002, loaded2.Protocols.Hysteria2.Port)
}

func TestFileRepository_Save_NilConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	repo := NewFileRepository(configPath, nil, zap.NewNop())
	ctx := context.Background()

	err := repo.Save(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "配置對象為空")
}

func TestFileRepository_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	repo := NewFileRepository(configPath, nil, zap.NewNop())
	ctx := context.Background()

	cfg := domainConfig.DefaultConfig()
	cfg.Protocols.TUIC.Enabled = true

	// 保存
	err := repo.Save(ctx, cfg)
	require.NoError(t, err)

	// 验证没有临时文件残留
	files, _ := os.ReadDir(tmpDir)
	tmpFiles := 0
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".tmp" {
			tmpFiles++
		}
	}
	assert.Equal(t, 0, tmpFiles, "不应有临时文件残留")
}

func TestFileRepository_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	repo := NewFileRepository(configPath, nil, zap.NewNop())
	ctx := context.Background()

	// 初始化配置
	cfg := domainConfig.DefaultConfig()
	repo.Save(ctx, cfg)

	// 并发读取
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := repo.Load(ctx)
			assert.NoError(t, err)
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestFileRepository_DeepCopy(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	repo := NewFileRepository(configPath, nil, zap.NewNop())
	ctx := context.Background()

	// 保存配置
	cfg := domainConfig.DefaultConfig()
	cfg.Protocols.Hysteria2.Enabled = true
	cfg.Protocols.Hysteria2.Port = 12345
	repo.Save(ctx, cfg)

	// 第一次加载
	cfg1, err := repo.Load(ctx)
	require.NoError(t, err)

	// 第二次加载
	cfg2, err := repo.Load(ctx)
	require.NoError(t, err)

	// 修改 cfg1
	cfg1.Protocols.Hysteria2.Port = 99999

	// cfg2 不应受影响
	assert.Equal(t, 12345, cfg2.Protocols.Hysteria2.Port, "深拷贝失败：cfg2 受到了 cfg1 的影响")
}

func TestFileRepository_MultipleFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	repo := NewFileRepository(configPath, nil, zap.NewNop())
	ctx := context.Background()

	// 设置多个字段
	cfg := domainConfig.DefaultConfig()
	cfg.Certificate.DNSProvider = "cloudflare"
	cfg.Certificate.DNSProviderID = "test-id"
	cfg.Certificate.DNSProviderSecret = "test-secret"
	cfg.Protocols.Hysteria2.Enabled = true
	cfg.Protocols.TUIC.Enabled = false

	// 保存
	err := repo.Save(ctx, cfg)
	require.NoError(t, err)

	// 加载并验证所有字段
	loaded, err := repo.Load(ctx)
	require.NoError(t, err)

	assert.Equal(t, "cloudflare", loaded.Certificate.DNSProvider)
	assert.Equal(t, "test-id", loaded.Certificate.DNSProviderID)
	assert.Equal(t, "test-secret", loaded.Certificate.DNSProviderSecret)
	assert.True(t, loaded.Protocols.Hysteria2.Enabled)
	assert.False(t, loaded.Protocols.TUIC.Enabled)
}

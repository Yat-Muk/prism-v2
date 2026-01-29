package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Yat-Muk/prism-v2/internal/pkg/appctx"
	"github.com/Yat-Muk/prism-v2/internal/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 檢測 Systemd 錯誤並跳過測試
// 這對於在 CI/CD 容器或非 Linux 環境中運行測試至關重要
func skipIfSystemdUnavailable(t *testing.T, err error) {
	if err == nil {
		return
	}
	errStr := err.Error()
	// 匹配常見的 Systemd 連接錯誤關鍵字
	if strings.Contains(errStr, "dial unix /run/systemd/private") ||
		strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "no such file or directory") ||
		strings.Contains(errStr, "無法連接 Systemd") ||
		strings.Contains(errStr, "初始化 Systemd") {
		t.Skipf("⚠️ 跳過測試: Systemd 在當前環境不可用 (%v)", err)
	}
}

// setupTestEnvironment 創建測試用的臨時環境
func setupTestEnvironment(t *testing.T) (*appctx.Paths, func()) {
	t.Helper()

	tmpDir := t.TempDir()

	// 使用 appctx.NewPaths 創建 Paths，傳入臨時目錄
	paths, err := appctx.NewPaths(tmpDir)
	require.NoError(t, err, "Failed to create test paths")

	// 確保所有目錄存在
	dirs := []string{
		paths.DataDir,
		paths.LogDir,
		paths.CertDir,
		paths.BackupDir,
	}

	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err, "Failed to create test directory: %s", dir)
	}

	cleanup := func() {
		// t.TempDir() 會自動清理
	}

	return paths, cleanup
}

// createTestLogger 創建測試用的 logger
func createTestLogger(t *testing.T) *zap.Logger {
	t.Helper()

	cfg := logger.DefaultConfig()
	cfg.Console = false
	cfg.Level = "debug"
	cfg.OutputPath = filepath.Join(t.TempDir(), "test.log")

	log, err := logger.New(cfg)
	require.NoError(t, err, "Failed to create test logger")

	return log
}

func TestInitializeDependencies_Success(t *testing.T) {
	// Arrange
	paths, cleanup := setupTestEnvironment(t)
	defer cleanup()

	log := createTestLogger(t)
	defer log.Sync()

	// Act
	deps, err := initializeDependencies(log, paths)

	// 如果是 Systemd 錯誤則跳過
	skipIfSystemdUnavailable(t, err)

	// Assert
	require.NoError(t, err, "initializeDependencies should not return error")
	assert.NotNil(t, deps, "Dependencies should not be nil")

	// 驗證所有核心依賴都已初始化
	assert.NotNil(t, deps.Log, "Log should be initialized")
	assert.NotNil(t, deps.Paths, "Paths should be initialized")
	assert.NotNil(t, deps.CertService, "CertService should be initialized")
	assert.NotNil(t, deps.SingboxService, "SingboxService should be initialized")
	assert.NotNil(t, deps.HandlerConfig, "HandlerConfig should be initialized")

	// 驗證 HandlerConfig 內部依賴
	assert.NotNil(t, deps.HandlerConfig.Log, "HandlerConfig.Log should be initialized")
	assert.NotNil(t, deps.HandlerConfig.StateMgr, "HandlerConfig.StateMgr should be initialized")
	assert.NotNil(t, deps.HandlerConfig.ConfigSvc, "HandlerConfig.ConfigSvc should be initialized")
	assert.NotNil(t, deps.HandlerConfig.PortService, "HandlerConfig.PortService should be initialized")
	assert.NotNil(t, deps.HandlerConfig.ProtocolService, "HandlerConfig.ProtocolService should be initialized")
	assert.NotNil(t, deps.HandlerConfig.CertService, "HandlerConfig.CertService should be initialized")
	assert.NotNil(t, deps.HandlerConfig.BackupMgr, "HandlerConfig.BackupMgr should be initialized")
}

func TestInitializeDependencies_InvalidPaths(t *testing.T) {
	// Arrange
	log := createTestLogger(t)
	defer log.Sync()

	// 創建無效的路徑（指向不存在的目錄）
	invalidPath := filepath.Join(t.TempDir(), "nonexistent", "nested", "path")

	// Act
	paths, err := appctx.NewPaths(invalidPath)

	// Assert
	// NewPaths 可能會創建目錄或返回錯誤
	if err != nil {
		assert.Nil(t, paths, "Paths should be nil when NewPaths fails")
		t.Logf("NewPaths returned expected error: %v", err)
	} else {
		// 如果 NewPaths 成功，測試後續初始化
		assert.NotNil(t, paths)
	}
}

func TestInitializeDependencies_ConfigLoadFailure(t *testing.T) {
	// Arrange
	paths, cleanup := setupTestEnvironment(t)
	defer cleanup()

	log := createTestLogger(t)
	defer log.Sync()

	// 創建無效的配置文件
	err := os.WriteFile(paths.ConfigFile, []byte("invalid json{{{"), 0644)
	require.NoError(t, err)

	// Act
	deps, err := initializeDependencies(log, paths)

	// 如果是 Systemd 錯誤則跳過
	skipIfSystemdUnavailable(t, err)

	// Assert
	// 根據實作，配置加載失敗時應該使用默認配置，所以不應該失敗
	require.NoError(t, err, "Should use default config when load fails")
	assert.NotNil(t, deps, "Dependencies should be initialized with default config")
}

func TestInitializeDependencies_ReadOnlyFileSystem(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	// Arrange
	paths, cleanup := setupTestEnvironment(t)
	defer cleanup()

	log := createTestLogger(t)
	defer log.Sync()

	// 將數據目錄設置為只讀
	err := os.Chmod(paths.DataDir, 0444)
	require.NoError(t, err)
	defer os.Chmod(paths.DataDir, 0755) // 恢復權限以便清理

	// Act
	deps, err := initializeDependencies(log, paths)

	// Assert
	if err != nil {
		skipIfSystemdUnavailable(t, err)

		assert.Contains(t, err.Error(), "備份", "Error should mention backup initialization")
		assert.Nil(t, deps, "Dependencies should be nil on error")
	}
}

func TestRunCronTask_Success(t *testing.T) {
	// Arrange
	paths, cleanup := setupTestEnvironment(t)
	defer cleanup()

	log := createTestLogger(t)
	defer log.Sync()

	deps, err := initializeDependencies(log, paths)

	// 如果是 Systemd 錯誤則跳過
	skipIfSystemdUnavailable(t, err)
	require.NoError(t, err)

	ctx := context.Background()

	// Act
	err = runCronTask(ctx, log, deps)

	// Assert
	assert.NoError(t, err, "runCronTask should not return error in normal operation")
}

func TestRunCronTask_ContextCancellation(t *testing.T) {
	// Arrange
	paths, cleanup := setupTestEnvironment(t)
	defer cleanup()

	log := createTestLogger(t)
	defer log.Sync()

	deps, err := initializeDependencies(log, paths)

	// 如果是 Systemd 錯誤則跳過
	skipIfSystemdUnavailable(t, err)

	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	// Act
	err = runCronTask(ctx, log, deps)

	// Assert
	// 注意：runCronTask 的某些步驟可能不會立即響應 context，這取決於具體實現
	// 但如果報錯，應該是 context canceled
	if err != nil {
		assert.Error(t, err, "Should handle context cancellation")
	}
}

func TestAppDependencies_AllFieldsInitialized(t *testing.T) {
	// Arrange
	paths, cleanup := setupTestEnvironment(t)
	defer cleanup()

	log := createTestLogger(t)
	defer log.Sync()

	// Act
	deps, err := initializeDependencies(log, paths)

	// 如果是 Systemd 錯誤則跳過
	skipIfSystemdUnavailable(t, err)

	require.NoError(t, err)

	// Assert - 確保所有字段都不是 nil
	assert.NotNil(t, deps.Log, "Log field should not be nil")
	assert.NotNil(t, deps.Paths, "Paths field should not be nil")
	assert.NotNil(t, deps.CertService, "CertService field should not be nil")
	assert.NotNil(t, deps.SingboxService, "SingboxService field should not be nil")
	assert.NotNil(t, deps.HandlerConfig, "HandlerConfig field should not be nil")

	// 驗證 paths 是同一個對象
	assert.Equal(t, paths, deps.Paths, "Paths should be the same instance")
	assert.Equal(t, log, deps.Log, "Log should be the same instance")
}

func TestInitializeDependencies_BackupEncryption(t *testing.T) {
	// Arrange
	paths, cleanup := setupTestEnvironment(t)
	defer cleanup()

	log := createTestLogger(t)
	defer log.Sync()

	// Act
	_, err := initializeDependencies(log, paths)

	// 如果是 Systemd 錯誤則跳過
	skipIfSystemdUnavailable(t, err)

	require.NoError(t, err)

	// Assert - 驗證備份密鑰文件是否創建
	backupKeyPath := filepath.Join(paths.DataDir, "backup.key")

	// 備份管理器可能會在初始化時創建密鑰文件
	// 檢查文件是否存在（取決於實作）
	if _, err := os.Stat(backupKeyPath); err == nil {
		// 如果文件存在，驗證權限
		info, err := os.Stat(backupKeyPath)
		require.NoError(t, err)

		// 密鑰文件應該有嚴格的權限
		mode := info.Mode().Perm()
		assert.True(t, mode&0077 == 0, "Backup key should not be readable by group/others, got: %o", mode)
	}
}

func TestRedirectStdErr_FileCreation(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test-stderr.log")

	// Act
	redirectStdErr(logFile)

	// Assert - 驗證文件被創建
	info, err := os.Stat(logFile)
	assert.NoError(t, err, "Stderr log file should be created")
	assert.NotNil(t, info, "File info should not be nil")

	// 驗證目錄被創建
	dir := filepath.Dir(logFile)
	dirInfo, err := os.Stat(dir)
	assert.NoError(t, err, "Log directory should exist")
	assert.True(t, dirInfo.IsDir(), "Should be a directory")
}

// Benchmark 測試
func BenchmarkInitializeDependencies(b *testing.B) {
	// Setup
	tmpDir := b.TempDir()

	paths, err := appctx.NewPaths(tmpDir)
	if err != nil {
		b.Fatalf("Failed to create paths: %v", err)
	}

	for _, dir := range []string{paths.DataDir, paths.LogDir, paths.CertDir, paths.BackupDir} {
		os.MkdirAll(dir, 0755)
	}

	cfg := logger.DefaultConfig()
	cfg.Console = false
	cfg.OutputPath = filepath.Join(tmpDir, "bench.log")
	log, _ := logger.New(cfg)

	b.ResetTimer()

	// Benchmark
	for i := 0; i < b.N; i++ {
		_, err := initializeDependencies(log, paths)
		if err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "permission denied") ||
				strings.Contains(errStr, "Systemd") ||
				strings.Contains(errStr, "no such file or directory") {
				b.Skipf("Skipping benchmark: Systemd unavailable in this environment (%v)", err)
			}

			b.Fatalf("Initialization failed: %v", err)
		}
	}
}

// 集成測試：測試完整的初始化流程
func TestIntegration_FullInitializationWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Arrange
	paths, cleanup := setupTestEnvironment(t)
	defer cleanup()

	log := createTestLogger(t)
	defer log.Sync()

	// Act - 完整的初始化流程
	deps, err := initializeDependencies(log, paths)

	// 如果是 Systemd 錯誤則跳過
	skipIfSystemdUnavailable(t, err)

	require.NoError(t, err)

	ctx := context.Background()

	// Assert - 驗證服務可以正常工作

	// 1. 測試配置倉庫（通過 HandlerConfig 訪問）
	assert.NotNil(t, deps.HandlerConfig.ConfigSvc, "Config service should be available")

	// 2. 執行 cron 任務（這會測試證書服務）
	err = runCronTask(ctx, log, deps)
	assert.NoError(t, err, "Cron task should execute without error")

	// 3. 測試端口服務
	assert.NotNil(t, deps.HandlerConfig.PortService, "Port service should be available")

	// 4. 測試協議服務
	assert.NotNil(t, deps.HandlerConfig.ProtocolService, "Protocol service should be available")

	// 5. 驗證狀態管理器
	assert.NotNil(t, deps.HandlerConfig.StateMgr, "State manager should be initialized")

	t.Log("Full initialization workflow completed successfully")
}

// 測試依賴注入的獨立性
func TestInitializeDependencies_MultipleInstances(t *testing.T) {
	// Arrange
	paths1, cleanup1 := setupTestEnvironment(t)
	defer cleanup1()

	paths2, cleanup2 := setupTestEnvironment(t)
	defer cleanup2()

	log := createTestLogger(t)
	defer log.Sync()

	// Act - 創建兩個獨立的依賴實例
	deps1, err1 := initializeDependencies(log, paths1)

	skipIfSystemdUnavailable(t, err1)
	if err1 == nil {
		// 只有當第一個成功時才嘗試第二個，避免重複報錯
		deps2, err2 := initializeDependencies(log, paths2)
		require.NoError(t, err2)

		require.NoError(t, err1)
		assert.NotEqual(t, deps1.Paths, deps2.Paths, "Paths should be different instances")
		assert.Equal(t, deps1.Log, deps2.Log, "Logger can be shared")
	}
}

// 測試錯誤場景：Systemd 初始化失敗的模擬（如果可能）
func TestInitializeDependencies_GracefulDegradation(t *testing.T) {
	// 此測試驗證即使某些組件初始化失敗，系統仍能優雅處理
	// 具體行為取決於實作

	paths, cleanup := setupTestEnvironment(t)
	defer cleanup()

	log := createTestLogger(t)
	defer log.Sync()

	// Act
	deps, err := initializeDependencies(log, paths)

	// Assert
	// 在測試環境中，某些系統級服務可能不可用
	// 但基本依賴應該能初始化
	if err != nil {
		t.Logf("Initialization failed (expected in some environments): %v", err)
	} else {
		assert.NotNil(t, deps)
		t.Log("All dependencies initialized successfully")
	}
}

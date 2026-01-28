package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/pkg/crypto"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// FileRepository 基於文件的配置倉庫實現
type FileRepository struct {
	filePath     string
	mu           sync.RWMutex
	fileMu       sync.Mutex // 用於文件 I/O 的互斥鎖
	encryptor    *crypto.Encryptor
	logger       *zap.Logger
	cachedConfig *domainConfig.Config
	lastModTime  time.Time
}

func NewFileRepository(path string, encryptor *crypto.Encryptor, logger *zap.Logger) *FileRepository {
	return &FileRepository{
		filePath:  path,
		encryptor: encryptor,
		logger:    logger,
	}
}

// Load 加載配置（支持緩存、熱重載與自動解密）
func (r *FileRepository) Load(ctx context.Context) (*domainConfig.Config, error) {
	// =================================================================
	// 階段 1: 快速路徑 (Fast Path) - 嘗試讀取緩存
	// =================================================================
	r.mu.RLock()
	stat, err := os.Stat(r.filePath)

	// 情況 A: 文件不存在 -> 返回默認配置
	// 這是為了首次啟動或文件被刪除時的容錯
	if os.IsNotExist(err) {
		r.mu.RUnlock()
		r.logger.Info("配置文件不存在，初始化默認配置")
		return domainConfig.DefaultConfig(), nil
	}

	// 情況 B: 獲取文件信息失敗 (如權限問題)
	if err != nil {
		r.mu.RUnlock()
		return nil, fmt.Errorf("檢查配置文件狀態失敗: %w", err)
	}

	// 情況 C: 緩存命中 (Cache Hit)
	// 條件：緩存存在 且 文件修改時間未變
	if r.cachedConfig != nil && !stat.ModTime().After(r.lastModTime) {
		// ⚠️ 關鍵：必須返回深拷貝！
		// 如果直接返回 r.cachedConfig，外部對配置的修改會直接污染緩存
		cfg := r.cachedConfig.DeepCopy()
		r.mu.RUnlock()
		r.logger.Debug("配置未變更，使用內存緩存")
		return cfg, nil
	}
	r.mu.RUnlock()

	// =================================================================
	// 階段 2: 慢速路徑 (Slow Path) - 從磁盤重新加載
	// =================================================================
	r.mu.Lock()
	defer r.mu.Unlock()

	// 🔒 雙重檢查鎖定 (Double-Check Locking)
	// 在我們從 RUnlock 切換到 Lock 的空檔期，可能有另一個協程已經完成了加載。
	// 所以必須再次檢查文件狀態，避免重複 I/O。
	stat, err = os.Stat(r.filePath)
	if os.IsNotExist(err) {
		return domainConfig.DefaultConfig(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("檢查配置文件狀態失敗: %w", err)
	}
	if r.cachedConfig != nil && !stat.ModTime().After(r.lastModTime) {
		return r.cachedConfig.DeepCopy(), nil
	}

	// 1. 讀取文件內容
	// 這裡使用 r.fileMu 主要是為了防止和 Save 操作發生底層 I/O 衝突（儘管 Atomic Write 已減輕此風險）
	r.fileMu.Lock()
	content, err := os.ReadFile(r.filePath)
	r.fileMu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("讀取配置文件失敗: %w", err)
	}

	// 2. 解析 YAML
	cfg := &domainConfig.Config{}
	if err := yaml.Unmarshal(content, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件格式失敗: %w", err)
	}

	// 3. 解密敏感字段
	// 磁盤上的數據可能是加密的，加載到內存後需要解密供程序使用
	if r.encryptor != nil {
		if err := cfg.DecryptSensitiveFields(r.encryptor); err != nil {
			// 如果解密失敗（例如密鑰變更），記錄錯誤但暫不崩潰，便於排查
			r.logger.Error("配置解密失敗，部分字段可能無法使用", zap.Error(err))
			return nil, fmt.Errorf("解密敏感配置失敗: %w", err)
		}
	}

	// 4. 更新緩存
	// 緩存一份乾淨的、已解密的副本
	r.cachedConfig = cfg.DeepCopy()
	r.lastModTime = stat.ModTime()

	r.logger.Info("配置文件已從磁盤重新加載",
		zap.String("path", r.filePath),
		zap.Time("mod_time", r.lastModTime),
	)

	// 5. 返回副本
	return cfg, nil
}

// Save 保存配置到文件（原子寫入）
func (r *FileRepository) Save(ctx context.Context, cfg *domainConfig.Config) error {
	if cfg == nil {
		return fmt.Errorf("配置對象為空")
	}

	r.fileMu.Lock()
	defer r.fileMu.Unlock()

	// 1. 深拷貝配置（避免修改內存中的原始對象影響其他協程）
	cfgCopy := cfg.DeepCopy()

	// 2. 加密敏感字段
	if r.encryptor != nil {
		if err := cfgCopy.EncryptSensitiveFields(r.encryptor); err != nil {
			return fmt.Errorf("加密配置失敗: %w", err)
		}
	}

	// 3. 序列化
	data, err := yaml.Marshal(cfgCopy)
	if err != nil {
		return fmt.Errorf("序列化配置失敗: %w", err)
	}

	// 4. 原子寫入 (Atomic Write)
	// 步驟：創建臨時文件 -> 寫入數據 -> Sync -> 關閉 -> Rename
	dir := filepath.Dir(r.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("創建配置目錄失敗: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, "config.*.yaml.tmp")
	if err != nil {
		return fmt.Errorf("創建臨時文件失敗: %w", err)
	}
	tmpName := tmpFile.Name()

	// 確保在出錯時清理臨時文件
	writeSuccess := false
	defer func() {
		if !writeSuccess {
			tmpFile.Close()
			os.Remove(tmpName)
		}
	}()

	// 寫入數據
	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("寫入數據失敗: %w", err)
	}

	// 強制落盤
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("同步磁盤失敗: %w", err)
	}

	// 關閉文件
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("關閉臨時文件失敗: %w", err)
	}

	// 原子重命名
	if err := os.Rename(tmpName, r.filePath); err != nil {
		return fmt.Errorf("替換配置文件失敗: %w", err)
	}

	// 設置權限 (600 - 僅所有者可讀寫)
	if err := os.Chmod(r.filePath, 0600); err != nil {
		r.logger.Warn("設置文件權限失敗", zap.Error(err))
	}

	writeSuccess = true

	// 5. 更新緩存
	r.mu.Lock()
	r.cachedConfig = cfg.DeepCopy() // 緩存未加密的原始版本
	if stat, err := os.Stat(r.filePath); err == nil {
		r.lastModTime = stat.ModTime()
	}
	r.mu.Unlock()

	return nil
}

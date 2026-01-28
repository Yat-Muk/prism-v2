package application

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/Yat-Muk/prism-v2/internal/domain/config"
)

// ConfigService 配置服務
type ConfigService struct {
	repo     config.Repository
	migrator *config.Migrator
	keyGen   *config.KeyGenerator
	logger   *zap.Logger
	mu       sync.Mutex
}

// NewConfigService 創建配置服務
func NewConfigService(
	repo config.Repository,
	logger *zap.Logger,
) *ConfigService {
	return &ConfigService{
		repo:     repo,
		migrator: config.NewMigrator(),
		keyGen:   config.NewKeyGenerator(),
		logger:   logger,
	}
}

// ==========================================
// 核心讀寫方法 (滿足測試需求)
// ==========================================

// GetConfig 獲取當前配置
func (s *ConfigService) GetConfig(ctx context.Context) (*config.Config, error) {
	return s.repo.Load(ctx)
}

// UpdateConfig 原子更新配置
// 邏輯：Lock -> Load -> DeepCopy -> Modify -> Validate -> Save -> Unlock
func (s *ConfigService) UpdateConfig(ctx context.Context, modifier func(*config.Config) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. 加載當前配置
	currentCfg, err := s.repo.Load(ctx)
	if err != nil {
		return fmt.Errorf("加載配置失敗: %w", err)
	}

	// 2. 創建副本 (DeepCopy) 以免修改原始對象
	// 使用 YAML 序列化策略確保安全
	newCfg := currentCfg.DeepCopy()

	// 3. 應用修改函數
	if err := modifier(newCfg); err != nil {
		return fmt.Errorf("應用配置修改失敗: %w", err)
	}

	// 4. 驗證新配置
	if err := newCfg.Validate(); err != nil {
		return fmt.Errorf("新配置驗證失敗: %w", err)
	}

	// 5. 保存配置
	if err := s.repo.Save(ctx, newCfg); err != nil {
		return fmt.Errorf("保存配置失敗: %w", err)
	}

	s.logger.Info("配置已更新並保存")
	return nil
}

// ==========================================
// 業務邏輯方法 (保留原有功能)
// ==========================================

// LoadWithMigration 加載配置並自動遷移
func (s *ConfigService) LoadWithMigration(ctx context.Context) (*config.Config, error) {
	// 加載時也加鎖，防止遷移過程中被並發修改
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. 加載原始配置
	cfg, err := s.repo.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("加載配置失敗: %w", err)
	}

	cfg.FillDefaults()

	// 2. 檢查是否需要遷移
	if !s.migrator.NeedsMigration(cfg) {
		s.logger.Info("配置已是最新版本", zap.Int("version", cfg.Version))
		return cfg, nil
	}

	// 3. 執行遷移
	oldVersion := cfg.Version
	migrationPath := s.migrator.GetMigrationPath(oldVersion)
	s.logger.Info("開始配置遷移",
		zap.Int("from_version", oldVersion),
		zap.Int("to_version", config.ConfigVersionLatest),
		zap.String("migration_path", migrationPath),
	)

	newCfg, err := s.migrator.MigrateToLatest(cfg)
	if err != nil {
		return nil, fmt.Errorf("遷移失敗: %w", err)
	}

	// 4. 保存遷移後的配置
	if err := s.repo.Save(ctx, newCfg); err != nil {
		return nil, fmt.Errorf("保存遷移後配置失敗: %w", err)
	}

	s.logger.Info("配置遷移完成",
		zap.Int("old_version", oldVersion),
		zap.Int("new_version", newCfg.Version),
	)

	return newCfg, nil
}

// RegenerateRealityKeys 重新生成 Reality 密鑰
func (s *ConfigService) RegenerateRealityKeys(ctx context.Context, cfg *config.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	keypair, err := s.keyGen.GenerateRealityKeypair()
	if err != nil {
		return fmt.Errorf("生成 Reality 密鑰失敗: %w", err)
	}

	shortID, err := s.keyGen.GenerateShortID()
	if err != nil {
		return fmt.Errorf("生成 Short ID 失敗: %w", err)
	}

	// 更新所有 Reality 協議的密鑰
	if cfg.Protocols.RealityVision.Enabled {
		cfg.Protocols.RealityVision.PublicKey = keypair.PublicKey
		cfg.Protocols.RealityVision.ShortID = shortID
	}

	if cfg.Protocols.RealityGRPC.Enabled {
		cfg.Protocols.RealityGRPC.PublicKey = keypair.PublicKey
		cfg.Protocols.RealityGRPC.ShortID = shortID
	}

	if cfg.Protocols.AnyTLSReality.Enabled {
		cfg.Protocols.AnyTLSReality.PublicKey = keypair.PublicKey
		cfg.Protocols.AnyTLSReality.ShortID = shortID
	}

	cfg.FillDefaults()

	// 保存配置
	if err := s.repo.Save(ctx, cfg); err != nil {
		return fmt.Errorf("保存配置失敗: %w", err)
	}

	s.logger.Info("Reality 密鑰已重新生成", zap.String("public_key", keypair.PublicKey[:16]+"..."))
	return nil
}

// SaveWithDefaults 保存配置並自動填充默認值
func (s *ConfigService) SaveWithDefaults(ctx context.Context, cfg *config.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg.FillDefaults()
	return s.repo.Save(ctx, cfg)
}

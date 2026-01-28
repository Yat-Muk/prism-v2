package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Yat-Muk/prism-v2/internal/application"
	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/domain/protocol"
	domainSingbox "github.com/Yat-Muk/prism-v2/internal/domain/singbox"
	"github.com/Yat-Muk/prism-v2/internal/infra/acme"
	"github.com/Yat-Muk/prism-v2/internal/infra/backup"
	"github.com/Yat-Muk/prism-v2/internal/infra/certinfo"
	infraConfig "github.com/Yat-Muk/prism-v2/internal/infra/config"
	"github.com/Yat-Muk/prism-v2/internal/infra/firewall"
	infraSingbox "github.com/Yat-Muk/prism-v2/internal/infra/singbox"
	infraSystem "github.com/Yat-Muk/prism-v2/internal/infra/system"
	"github.com/Yat-Muk/prism-v2/internal/pkg/appctx"
	"github.com/Yat-Muk/prism-v2/internal/pkg/cert"
	"github.com/Yat-Muk/prism-v2/internal/pkg/version"
	"github.com/Yat-Muk/prism-v2/internal/tui/handlers"
	"github.com/Yat-Muk/prism-v2/internal/tui/state"
	"go.uber.org/zap"
)

type AppDependencies struct {
	Log            *zap.Logger
	Paths          *appctx.Paths
	CertService    *application.CertService
	SingboxService *application.SingboxService
	HandlerConfig  *handlers.Config
}

func initializeDependencies(log *zap.Logger, paths *appctx.Paths) (*AppDependencies, error) {
	// ==========================================
	// 1. 基礎設施層 (Infrastructure Layer)
	// ==========================================

	sysInfo := infraSystem.NewSystemInfo(log)
	executor := infraSystem.NewExecutor(log)

	systemdMgr, err := infraSystem.NewSystemdManager(log)
	if err != nil {
		return nil, fmt.Errorf("初始化 Systemd 管理器失敗: %w", err)
	}

	backupKeyPath := filepath.Join(paths.DataDir, "backup.key")
	backupMgr, err := backup.NewManager(paths.BackupDir, backupKeyPath, backup.RetentionPolicy{
		MaxFiles: 5,
	})
	if err != nil {
		return nil, fmt.Errorf("備份管理器初始化失敗: %w", err)
	}

	configRepo := infraConfig.NewFileRepository(paths.ConfigFile, nil, log)
	firewallMgr := firewall.NewManager(log)

	acmeSvc, err := acme.NewService(paths, "", log)
	if err != nil {
		return nil, fmt.Errorf("初始化 ACME 服務失敗: %w", err)
	}

	protoFactory := protocol.NewFactory(paths)

	certRepo := certinfo.NewRepository(paths.CertDir)

	// ==========================================
	// 2. 加載初始配置
	// ==========================================
	initialConfig, err := configRepo.Load(context.Background())
	if err != nil {
		log.Warn("加載配置失敗，使用默認值", zap.Error(err))
		initialConfig = domainConfig.DefaultConfig()
	}

	// ==========================================
	// 3. 應用服務層 (Application Layer)
	// ==========================================

	configSvc := application.NewConfigService(configRepo, log)
	if err := configSvc.SaveWithDefaults(context.Background(), initialConfig); err != nil {
		log.Warn("初始化保存配置失敗", zap.Error(err))
	}

	portSvc := application.NewPortService(log)
	protocolSvc := application.NewProtocolService(log)
	selfSignedGen := cert.NewSelfSignedGenerator(log)

	// 參數順序：paths, certRepo, acmeSvc, selfSignedGen, configSvc, log
	certSvc := application.NewCertService(paths, certRepo, acmeSvc, selfSignedGen, configSvc, log)

	// Singbox Service
	sbGenerator := domainSingbox.NewGenerator("unknown", protoFactory)
	sbInfraService := infraSingbox.NewService(systemdMgr, log, firewallMgr, paths)
	singboxSvc := application.NewSingboxService(sbGenerator, sbInfraService, firewallMgr, paths, log)

	// ==========================================
	// 4. 狀態管理 (State Management)
	// ==========================================
	stateCfg := &state.Config{
		Log:             log,
		Paths:           paths,
		SysInfo:         sysInfo,
		SingboxService:  singboxSvc,
		ConfigRepo:      configRepo,
		InitialConfig:   initialConfig,
		PortService:     portSvc,
		ProtocolService: protocolSvc,
		CertService:     certSvc,
		BackupManager:   backupMgr,
		ScriptVersion:   version.Version,
	}
	stateMgr := state.NewManager(stateCfg)

	// ==========================================
	// 5. TUI Handler 配置
	// ==========================================
	handlerCfg := &handlers.Config{
		Log:             log,
		StateMgr:        stateMgr,
		ConfigSvc:       configSvc,
		PortService:     portSvc,
		ProtocolService: protocolSvc,
		SingboxService:  singboxSvc,
		CertService:     certSvc,
		BackupMgr:       backupMgr,
		SysInfo:         sysInfo,
		Paths:           paths,
		Executor:        executor,
		FirewallMgr:     firewallMgr,
		ProtoFactory:    protoFactory,
	}

	return &AppDependencies{
		Log:            log,
		Paths:          paths,
		CertService:    certSvc,
		SingboxService: singboxSvc,
		HandlerConfig:  handlerCfg,
	}, nil
}

func runCronTask(ctx context.Context, log *zap.Logger, deps *AppDependencies) error {
	log.Info("執行證書檢查...")
	renewed, err := deps.CertService.CheckAndRenewAll(ctx)
	if err != nil {
		log.Error("證書檢查部分失敗", zap.Error(err))
	}

	if renewed {
		log.Info("證書已更新，重啟核心服務...")
		if err := deps.SingboxService.Restart(ctx); err != nil {
			return err
		}
	} else {
		log.Debug("無證書需要更新")
	}

	return nil
}

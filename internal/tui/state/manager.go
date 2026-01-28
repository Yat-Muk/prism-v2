package state

import (
	"github.com/Yat-Muk/prism-v2/internal/application"
	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/infra/backup"
	"github.com/Yat-Muk/prism-v2/internal/infra/system"
	"github.com/Yat-Muk/prism-v2/internal/pkg/appctx"
	"go.uber.org/zap"
)

// Config 初始化配置
type Config struct {
	Log             *zap.Logger
	SysInfo         *system.SystemInfo
	SingboxService  *application.SingboxService
	ConfigRepo      domainConfig.Repository
	InitialConfig   *domainConfig.Config
	PortService     application.PortService
	ProtocolService application.ProtocolService
	CertService     *application.CertService
	BackupManager   *backup.Manager
	ScriptVersion   string
	Paths           *appctx.Paths
}

// Manager 狀態管理器 (State Container)
type Manager struct {
	log *zap.Logger

	// 各個子狀態模塊
	ui        *UIState
	system    *SystemState
	config    *ConfigState
	install   *InstallState
	port      *PortState
	cert      *CertState
	backup    *BackupState
	core      *CoreState
	routing   *RoutingState
	service   *ServiceState
	uninstall *UninstallState
	tools     *ToolsState
	logState  *LogState
	node      *NodeState

	paths *appctx.Paths
}

// NewManager 創建狀態管理器
func NewManager(cfg *Config) *Manager {
	m := &Manager{
		log:   cfg.Log,
		paths: cfg.Paths,
	}

	// 初始化各個子狀態
	m.ui = NewUIState()
	m.system = NewSystemState()
	m.config = NewConfigState(cfg.InitialConfig)
	m.install = NewInstallState()
	m.port = NewPortState()
	m.cert = NewCertState()
	m.backup = NewBackupState()
	m.core = NewCoreState(cfg.ScriptVersion)
	m.routing = NewRoutingState()
	m.service = NewServiceState()
	m.uninstall = NewUninstallState()
	m.tools = NewToolsState()
	m.logState = NewLogState()
	m.node = NewNodeState()

	return m
}

// Getters 訪問器

func (m *Manager) UI() *UIState               { return m.ui }
func (m *Manager) System() *SystemState       { return m.system }
func (m *Manager) Config() *ConfigState       { return m.config }
func (m *Manager) Install() *InstallState     { return m.install }
func (m *Manager) Port() *PortState           { return m.port }
func (m *Manager) Cert() *CertState           { return m.cert }
func (m *Manager) Backup() *BackupState       { return m.backup }
func (m *Manager) Core() *CoreState           { return m.core }
func (m *Manager) Routing() *RoutingState     { return m.routing }
func (m *Manager) Service() *ServiceState     { return m.service }
func (m *Manager) Uninstall() *UninstallState { return m.uninstall }
func (m *Manager) Tools() *ToolsState         { return m.tools }
func (m *Manager) Log() *LogState             { return m.logState }
func (m *Manager) Node() *NodeState           { return m.node }

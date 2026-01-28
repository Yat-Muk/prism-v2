package handlers

import (
	"github.com/Yat-Muk/prism-v2/internal/application"
	"github.com/Yat-Muk/prism-v2/internal/domain/protocol"
	"github.com/Yat-Muk/prism-v2/internal/infra/backup"
	"github.com/Yat-Muk/prism-v2/internal/infra/firewall"
	"github.com/Yat-Muk/prism-v2/internal/infra/system"
	"github.com/Yat-Muk/prism-v2/internal/pkg/appctx"
	"github.com/Yat-Muk/prism-v2/internal/tui/state"
	"go.uber.org/zap"
)

// Config 用於初始化 Handlers 的配置結構體
type Config struct {
	Log             *zap.Logger
	StateMgr        *state.Manager
	ConfigSvc       *application.ConfigService
	PortService     application.PortService
	ProtocolService application.ProtocolService
	SingboxService  *application.SingboxService
	CertService     *application.CertService
	BackupMgr       *backup.Manager
	SysInfo         *system.SystemInfo
	Paths           *appctx.Paths
	Executor        system.Executor
	FirewallMgr     firewall.Manager
	ProtoFactory    protocol.Factory
}

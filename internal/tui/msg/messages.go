package msg

import (
	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/tui/types"
)

// DataUpdateMsg 數據更新消息
type DataUpdateMsg struct {
	Stats         types.SystemStats
	ServiceStats  types.ServiceStats
	CoreVersion   string
	LatestVersion string
	HasUpdate     bool
	IsInstalled   bool
}

// ConfigUpdateMsg 配置更新消息
type ConfigUpdateMsg struct {
	Err       error
	NewConfig *domainConfig.Config
	Applied   bool
	Message   string
}

// ConfigLoadedMsg 配置加載消息
type ConfigLoadedMsg struct {
	Config *domainConfig.Config
	Err    error
	Silent bool
}

// InstallResultMsg 安裝結果消息
type InstallResultMsg struct {
	Err error
}

// ServiceResultMsg 服務操作結果消息
type ServiceResultMsg struct {
	Action string // "restart", "stop"
	Err    error
}

// CertRequestMsg 證書申請消息
type CertRequestMsg struct {
	Domain   string
	Provider string
	Err      error
}

// CertRenewMsg 證書續期消息
type CertRenewMsg struct {
	Domain string
	Err    error
}

// CertDeleteMsg 證書刪除消息
type CertDeleteMsg struct {
	Domain string
	Err    error
}

// CertListRefreshMsg 證書列表刷新消息
type CertListRefreshMsg struct {
	ACMEDomains       []string
	SelfSignedDomains []string
	CertList          []types.CertInfo
	CurrentCAProvider string
}

// ProviderSwitchMsg CA 切換消息
type ProviderSwitchMsg struct {
	Provider string
	Err      error
}

// RoutingConfigLoadedMsg 路由配置加載消息
type RoutingConfigLoadedMsg struct {
	Type   string
	Config interface{}
	Err    error
}

// BackupListMsg 備份列表消息
type BackupListMsg struct {
	Entries []types.BackupItem // ✅ 修正類型
	Err     error
}

// BackupCreateMsg 備份創建消息
type BackupCreateMsg struct {
	Name string
	Err  error
}

// BackupRestoreMsg 備份恢復消息
type BackupRestoreMsg struct {
	Err error
}

// SystemInfoLoadedMsg 系統信息加載消息 (通用)
type SystemInfoLoadedMsg struct {
	Type string // "fail2ban", "swap", "bbr"
	Data interface{}
	Err  error
}

// StreamingCheckResultMsg 流媒體檢測結果
type StreamingCheckResultMsg struct {
	Result *types.StreamingCheckResult
	Err    error
}

// LogInfoMsg 日誌信息消息
type LogInfoMsg struct {
	LogLevel   string
	LogPath    string
	LogSize    string
	TodayLines int
	ErrorCount int
	RecentLogs []string
	Err        error
}

// LogViewMsg 日誌內容查看消息
type LogViewMsg struct {
	Mode      string
	Logs      []string
	Err       error
	Following bool
}

// UUIDGeneratedMsg UUID 生成消息
type UUIDGeneratedMsg struct {
	UUID string
}

// SwapInfoMsg Swap 信息
type SwapInfoMsg struct {
	Info *types.SwapInfo
	Err  error
}

// Fail2BanInfoMsg Fail2Ban 信息
type Fail2BanInfoMsg struct {
	Info *types.Fail2BanInfo
	Err  error
}

// CleanupInfoMsg 清理信息
type CleanupInfoMsg struct {
	Info *types.CleanupInfo
	Err  error
}

// BBRInfoMsg BBR 信息
type BBRInfoMsg struct {
	Info *types.BBRInfo
	Err  error
}

// CoreCheckMsg 核心更新檢查結果
type CoreCheckMsg struct {
	HasUpdate     bool
	LatestVersion string
	IsSilent      bool
	Err           error
}

// CoreVersionsMsg 核心版本列表消息
type CoreVersionsMsg struct {
	Versions []string
	Err      error
}

// CoreInstallMsg 核心安裝消息
type CoreInstallMsg struct {
	Version   string
	Success   bool
	Message   string
	Installed bool
	Err       error
}

// ScriptCheckMsg 腳本更新檢查結果
type ScriptCheckMsg struct {
	Success   bool
	LatestVer string
	Changelog string
	Err       error
}

// ServiceHealthMsg 服務健康檢查消息
type ServiceHealthMsg struct {
	Result *types.HealthCheckResult
	Err    error
}

// ServiceAutoStartMsg 服務自啓動消息
type ServiceAutoStartMsg struct {
	Enabled bool
	Err     error
}

// NodeInfoMsg 節點信息消息
type NodeInfoMsg struct {
	Type         string
	Links        []types.ProtocolLink
	Subscription *types.SubscriptionInfo
	QRCode       string
	Content      string
	ClientConfig *types.ClientConfigInfo
	Err          error
	QRType       string
	Info         *types.NodeInfo
}

// NodeParamsDataMsg 節點參數數據消息
type NodeParamsDataMsg struct {
	ServerIP string
	Err      error
}

// UninstallInfoMsg 卸載信息消息
type UninstallInfoMsg struct {
	Info *types.UninstallInfo
	Err  error
}

// UninstallCompleteMsg 卸載完成消息
type UninstallCompleteMsg struct {
	Steps   []types.UninstallStep
	Success bool
	Err     error
}

// ViewChangeMsg 用於延遲或異步切換視圖
type ViewChangeMsg struct {
	// 此處依賴 state 包會導致循環引用，
	// 建議直接傳遞 int 視圖 ID，或者在 state 包中定義消息
	ViewID int
}

// CommandResultMsg 通用命令結果消息
type CommandResultMsg struct {
	Success bool
	Message string
	Data    string
	Err     error
}

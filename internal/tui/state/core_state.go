package state

// CoreState 核心服務狀態管理
type CoreState struct {
	// 版本信息 (直接導出)
	CoreVersion         string
	LatestVersion       string
	HasUpdate           bool
	IsInstalled         bool
	ScriptVersion       string
	ScriptLatestVersion string   // 腳本最新版本
	ScriptChangelog     string   // 更新日誌/更新內容
	IsCheckingScript    bool     // 用於顯示 loading 狀態
	UpdateSource        string   // "github", "ghproxy", "custom"
	AvailableVers       []string // 可用版本列表
	ConfirmAction       string   // "reinstall", "uninstall", "install_dev"
}

// NewCoreState 創建核心狀態管理器
func NewCoreState(scriptVer string) *CoreState {
	return &CoreState{
		CoreVersion:         "unknown",
		ScriptVersion:       scriptVer,
		UpdateSource:        "github",
		AvailableVers:       []string{},
		ScriptLatestVersion: "",
		ScriptChangelog:     "",
		IsCheckingScript:    false,
	}
}

// NeedsConfirmation 是否需要確認
func (s *CoreState) NeedsConfirmation() bool {
	return s.ConfirmAction != ""
}

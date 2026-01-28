package state

import "github.com/Yat-Muk/prism-v2/internal/tui/types"

// ToolsState 工具箱業務狀態
type ToolsState struct {
	// --- 流媒體檢測 ---
	IsCheckingStreaming bool
	StreamingResult     *types.StreamingCheckResult

	// --- 系統清理 ---
	CleanupInfo *types.CleanupInfo

	// --- Swap 管理 ---
	SwapInfo      *types.SwapInfo
	SwapInputMode bool

	// --- Fail2Ban ---
	Fail2BanInfo          *types.Fail2BanInfo // 信息
	Fail2BanInputMode     bool                // 是否正在輸入 IP (解封)
	Fail2BanConfigStep    int                 // 配置流程步驟 (0:無, 1:重試次數, 2:封禁時間)
	Fail2BanTempMaxRetry  string              // 臨時存儲用戶輸入的重試次數
	Fail2BanLogOutput     []string            // 封禁列表數據
	IsViewingFail2BanLogs bool                // 標記是否正在從 Fail2Ban 查看日誌/列表

	// --- BBR ---
	BBRInfo           *types.BBRInfo
	BBRInstallConfirm bool   // 通用的內核安裝確認模式
	BBRInstallTarget  string // "xanmod" 或 "bbr2"

	// --- Log 日誌 ---
	LogInfo *types.LogInfo
}

// NewToolsState 初始化
func NewToolsState() *ToolsState {
	return &ToolsState{
		IsCheckingStreaming:   false,
		SwapInputMode:         false,
		Fail2BanInputMode:     false,
		Fail2BanConfigStep:    0,
		IsViewingFail2BanLogs: false,
		BBRInstallConfirm:     false,
		BBRInstallTarget:      "",
		LogInfo:               nil,
	}
}

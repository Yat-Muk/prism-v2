package state

import (
	"github.com/Yat-Muk/prism-v2/internal/tui/types"
	"github.com/charmbracelet/bubbles/viewport"
)

// NodeState 負責存儲節點信息、鏈接和訂閱數據
type NodeState struct {
	// 基礎信息
	NodeInfo *types.NodeInfo

	// 協議鏈接列表
	Links []types.ProtocolLink

	// 訂閱信息
	Subscription *types.SubscriptionInfo

	// 列表選擇模式 ("links", "qrcode", "params")
	SelectionMode string

	// 存儲節點參數所需的數據
	ParamsIP string

	// 二維碼
	CurrentQRCode    string
	CurrentQRContent string
	CurrentQRType    string // "protocol" or "subscription"
	QRInvert         bool   // 是否反色
	QRLevel          string // 容錯率: "L", "M", "Q", "H"
	CurrentQRIndex   int    // 當前正在查看的鏈接索引 (用於刷新)

	// 客戶端配置導出信息
	ClientConfig *types.ClientConfigInfo

	// 視口組件
	Viewport      viewport.Model
	ViewportReady bool
}

func NewNodeState() *NodeState {
	return &NodeState{
		NodeInfo:       nil,
		Links:          []types.ProtocolLink{},
		Subscription:   nil,
		ViewportReady:  false,
		SelectionMode:  "links",
		QRInvert:       false,
		QRLevel:        "L",
		CurrentQRIndex: -1,
	}
}

func (n *NodeState) SetParamsData(ip string) {
	n.ParamsIP = ip
	n.SelectionMode = "params"

	// 重置滾動條位置，準備顯示新內容
	n.Viewport.GotoTop()
	n.ViewportReady = true
}

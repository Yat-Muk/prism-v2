package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/Yat-Muk/prism-v2/internal/tui/types"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderNodeInfo 渲染節點信息菜單
func RenderNodeInfo(info *types.NodeInfo, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("節點信息")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 查看和導出客戶端連接信息")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 服務器信息
	labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)

	var serverInfo string
	if info != nil {
		serverInfo = fmt.Sprintf("%s %s",
			labelStyle.Render(" 服務器地址："),
			valueStyle.Render(info.ServerIP))

		if len(info.Protocols) > 0 {
			serverInfo += fmt.Sprintf("\n%s %s",
				labelStyle.Render(" 已啟用協議："),
				valueStyle.Render(fmt.Sprintf("%d 個", len(info.Protocols))))
		}
	} else {
		serverInfo = lipgloss.NewStyle().
			Foreground(style.Muted).
			Render(" 正在加載節點信息...")
	}

	items := []MenuItem{
		{"", "", "", lipgloss.Color("")},
		{constants.KeyNode_Links, "查看協議鏈接", "(獲取協議的 URL 鏈接)", style.Aurora2},
		{constants.KeyNode_QRCode, "生成二維碼", "(可通過掃描二維碼導入)", style.StatusYellow},
		{constants.KeyNode_Subscription, "獲取訂閱鏈接", "(在線訂閱及 Base64 離線包)", style.StatusGreen},
		{constants.KeyNode_ClientConfig, "導出配置", "(導出完整配置文件/節點參數)", style.Snow1},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		serverInfo,
		menu,
		statusBlock,
		footer,
	)
}

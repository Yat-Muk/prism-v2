package view

import (
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderToolsMenu 渲染工具菜單
func RenderToolsMenu(cursor int, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("系統工具箱")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 系統管理、優化與安全工具集")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	items := []MenuItem{
		{constants.KeyTools_Streaming, "流媒體/IP 檢測", "(原生檢測/無外部依賴)", style.Aurora2},
		{constants.KeyTools_Swap, "虛擬內存 (SWAP)", "(弱雞適用)", style.Snow1},
		{constants.KeyTools_Fail2Ban, "Fail2Ban 防護", "(SSH 安全防護)", style.StatusGreen},
		{constants.KeyTools_TimeSync, "校准服務器時間", "(Asia/Shanghai)", style.Snow1},
		{constants.KeyTools_BBR, "BBR 加速與優化", "(原版BBR/XanMod-BBRv3)", style.Aurora3},
		{constants.KeyTools_Cleanup, "系統清理", "(清空日誌/緩存)", style.StatusYellow},
		{constants.KeyTools_Backup, "配置備份", "(導出密鑰與證書)", style.Snow1},
	}

	menu := renderMenuWithAlignment(items, cursor, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 選擇工具進行操作")

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		menu,
		"",
		instruction,
		statusBlock,
		footer,
	)
}

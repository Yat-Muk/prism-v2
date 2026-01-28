package view

import (
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/Yat-Muk/prism-v2/internal/tui/types"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderClientConfig 渲染客戶端配置導出
func RenderClientConfig(info *types.ClientConfigInfo, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("客戶端配置導出")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 導出完整的客戶端配置文件")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	items := []MenuItem{
		{constants.KeyExport_Full, "導出 sing-box 配置", "(包含所有設置的完整 JSON)", style.Aurora2},
		{constants.KeyExport_Clash, "導出 Clash.Meta 配置", "(適用於 Mihomo/Clash Verge)", style.StatusGreen},
		{constants.KeyExport_Custom, "節點參數", "(查看節點參數)", style.Snow1},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 配置文件保存在 /tmp 目錄，可通過 SCP 下載")

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

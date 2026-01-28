package view

import (
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderConfigMenu 渲染配置與協議菜單
func RenderConfigMenu(cursor int, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("配置與協議管理")

	desc1 := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 管理協議的開關、端口等核心配置，並提供配置界面")

	// 信息區：說明 + 灰色分隔線 + 當前值
	infoSep := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	infoBlock := lipgloss.JoinVertical(
		lipgloss.Left,
		desc1,
		infoSep,
	)

	items := []MenuItem{
		{constants.KeyConfig_Protocol, "協議開關管理", "(啟用/禁用各協議)", style.Aurora1},
		{constants.KeyConfig_SNI, "修改 SNI 域名", "(偽裝域名設置)", style.Snow1},
		{constants.KeyConfig_UUID, "修改 UUID", "(用戶標識符)", style.Snow1},
		{constants.KeyConfig_Port, "修改監聽端口", "(服務端口設置)", style.Snow1},
		{constants.KeyConfig_Padding, "AnyTLS 填充策略", "(調整偽裝流量特徵)", style.Snow1},

		{"", "", "", lipgloss.Color("")}, // 分組線

		{constants.KeyConfig_Reset, "重置 所有配置", "(恢復到初始狀態，請謹慎操作)", style.StatusRed},
	}

	menu := renderMenuWithAlignment(items, cursor, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 修改後必須「應用配置」才會生效")

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		infoBlock,
		menu,
		"",
		instruction,
		statusBlock,
		footer,
	)
}

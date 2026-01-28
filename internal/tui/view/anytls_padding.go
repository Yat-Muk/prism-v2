package view

import (
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderAnyTLSPaddingMenu 渲染 AnyTLS 填充策略菜單
// currentName 用來顯示「當前方案: xxx」
func RenderAnyTLSPaddingMenu(ti textinput.Model, currentName string, statusMsg string) string {
	header := renderSubpageHeader("AnyTLS 填充策略")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 選擇偽裝方案，模擬對應策略的真實流量特徵")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 當前策略: 灰色 + 藍色方案名
	labelStyle := lipgloss.NewStyle().Foreground(style.Snow2)
	nameStyle := lipgloss.NewStyle().Foreground(style.Aurora4)

	currentLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		labelStyle.Render(" 當前策略: "),
		nameStyle.Render(currentName),
	)

	items := []MenuItem{
		{"", "", "", lipgloss.Color("")},
		{constants.KeyPadding_Balanced, "均衡流", "(模擬網頁瀏覽，流量自然) [推薦]", style.Aurora1},
		{constants.KeyPadding_Minimal, "極簡流", "(省流/移動端，干擾最低)", style.Snow1},
		{constants.KeyPadding_HighResist, "高對抗流", "(針對被重度干擾時使用)", style.Snow1},
		{constants.KeyPadding_Video, "視頻特徵", "(模擬視頻啟播流量特徵)", style.Snow1},
		{constants.KeyPadding_Official, "官方默認", "(Sing-box 官方示例配置)", style.Snow1},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(` 
 提示： 
   • 會相應增加延遲和流量開銷
   • 理論上可將被識別概率降至極低`)

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		currentLine,
		menu,
		"",
		instruction,
		statusBlock,
		footer,
	)
}

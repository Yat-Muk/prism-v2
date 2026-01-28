package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderHy2PortMode 渲染 Hysteria 2 端口模式選擇頁
func RenderHy2PortMode(currentPort int, currentHopping string, ti textinput.Model, statusMsg string, isEditing bool) string {
	header := renderSubpageHeader("Hysteria 2 端口設置")

	desc1 := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 跳躍端口格式為 start-end, 例如: 20000-30000")

		// 信息顯示區域
	valStyle := lipgloss.NewStyle().Foreground(style.Aurora4)
	labelStyle := lipgloss.NewStyle().Foreground(style.Snow2)

	// 構建主端口行
	portBlock := lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render(" 當前主端口: "),
		valStyle.Render(fmt.Sprintf("%d", currentPort)),
	)

	// 構建跳躍端口行
	hoppingText := "未設置"
	if currentHopping != "" {
		hoppingText = currentHopping
	}
	hoppingBlock := lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render(" 當前跳躍端口: "),
		valStyle.Render(hoppingText),
	)

	// 分隔線
	infoSep := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 信息區塊組合
	infoBlock := lipgloss.JoinVertical(
		lipgloss.Left,
		desc1,
		infoSep,
		portBlock,
		hoppingBlock,
	)

	items := []MenuItem{
		{"", "", "", lipgloss.Color("")},
		{constants.KeyPort_Main, "修改主端口", "", style.Snow1},
		{constants.KeyPort_Hopping, "設置跳躍端口", "", style.Snow1},
		{"", "", "", lipgloss.Color("")},
		{constants.KeyPort_ClearHopping, "清除跳躍端口", "", style.StatusRed},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 端口跳躍範圍數建議不要超過 1000")

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

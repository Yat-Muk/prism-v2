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

// RenderLogMenu 渲染日誌管理菜單
func RenderLogMenu(info *types.LogInfo, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("日誌管理")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 查看和管理 sing-box 運行日誌")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 顯示日誌狀態
	labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)
	errorStyle := lipgloss.NewStyle().Foreground(style.StatusRed)

	var statusText string
	if info != nil {
		errorText := fmt.Sprintf("%d", info.ErrorCount)
		if info.ErrorCount > 0 {
			errorText = errorStyle.Render(errorText)
		} else {
			errorText = valueStyle.Render(errorText)
		}

		statusText = fmt.Sprintf(
			" %s %s\n %s %s\n %s %s\n %s %d 行\n %s %s",
			labelStyle.Render("日誌級別:"),
			valueStyle.Render(info.LogLevel),
			labelStyle.Render("日誌路徑:"),
			lipgloss.NewStyle().Foreground(style.Snow3).Render(info.LogPath),
			labelStyle.Render("文件大小:"),
			valueStyle.Render(info.LogSize),
			labelStyle.Render("今日日誌:"),
			info.TodayLines,
			labelStyle.Render("錯誤計數:"),
			errorText,
		)
	} else {
		statusText = lipgloss.NewStyle().
			Foreground(style.Muted).
			Render(" 加載日誌信息中...")
	}

	items := []MenuItem{
		{"", "", "", lipgloss.Color("")},
		{constants.KeyLog_Realtime, "查看實時日誌", "(顯示最新的日誌輸出)", style.Aurora1},
		{constants.KeyLog_Full, "查看完整日誌", "(顯示所有歷史日誌)", style.Snow1},
		{constants.KeyLog_Error, "查看錯誤日誌", "(僅顯示錯誤級別日誌)", style.StatusRed},
		{constants.KeyLog_Level, "修改日誌級別", "(Debug/Info/Warn/Error)", style.Snow1},
		{constants.KeyLog_Export, "導出日誌文件", "(保存到本地)", style.Snow1},
		{"", "", "", lipgloss.Color("")},
		{constants.KeyLog_Clear, "清空日誌", "(刪除現有日誌文件)", style.StatusRed},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		statusText,
		menu,
		statusBlock,
		footer,
	)
}

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

// RenderSystemCleanup 渲染系統清理頁面
func RenderSystemCleanup(info *types.CleanupInfo, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("系統清理")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 釋放磁盤空間，清理日誌與緩存")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 狀態顯示區域
	var statusText string
	if info != nil {
		labelStyle := lipgloss.NewStyle().Foreground(style.Snow3).Width(12)
		valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)
		totalStyle := lipgloss.NewStyle().Foreground(style.StatusOrange)

		// 使用表格佈局風格
		row1 := fmt.Sprintf(" %s%s", labelStyle.Render("系統日誌:"), valueStyle.Render(info.LogSize))
		row2 := fmt.Sprintf(" %s%s", labelStyle.Render("包緩存:"), valueStyle.Render(info.CacheSize))
		row3 := fmt.Sprintf(" %s%s", labelStyle.Render("臨時文件:"), valueStyle.Render(info.TempSize))
		row4 := fmt.Sprintf(" %s%s", labelStyle.Render("可清理總計:"), totalStyle.Render(info.TotalSize))

		statusText = lipgloss.JoinVertical(lipgloss.Left, row1, row2, row3, row4)
	} else {
		statusText = lipgloss.NewStyle().
			Foreground(style.Muted).
			Render(" 正在掃描系統空間...")
	}

	// 菜單選項
	items := []MenuItem{
		{"", "", "", lipgloss.Color("")},
		{constants.KeyCleanup_Scan, "重新掃描", "(刷新磁盤佔用數據)", style.Aurora2},
		{constants.KeyCleanup_Log, "清理日誌", "(清空系統與應用日誌)", style.Snow1},
		{constants.KeyCleanup_Pkg, "清理緩存", "(移除 apt/yum 下載包)", style.Snow1},
		{constants.KeyCleanup_Temp, "清理臨時", "(清空 /tmp 目錄)", style.Snow1},
		{"", "", "", lipgloss.Color("")},
		{constants.KeyCleanup_All, "一鍵清理", "(執行以上所有操作)", style.StatusRed},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	// 底部提示
	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 建議在服務停止或維護期間執行清理")

	statusBlock := RenderStatusMessage(statusMsg)
	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		statusText,
		menu,
		"",
		instruction,
		statusBlock,
		footer,
	)
}

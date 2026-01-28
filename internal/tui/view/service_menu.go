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

// RenderServiceMenu 渲染服務管理菜單
func RenderServiceMenu(
	serviceStats *types.ServiceStats,
	autoStart bool,
	ti textinput.Model,
	statusMsg string,
) string {
	header := renderSubpageHeader("服務管理")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 管理 sing-box 系統服務")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 顯示服務狀態
	labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	runningStyle := lipgloss.NewStyle().Foreground(style.StatusGreen).Bold(true)
	stoppedStyle := lipgloss.NewStyle().Foreground(style.StatusRed)
	enabledStyle := lipgloss.NewStyle().Foreground(style.Aurora2)
	disabledStyle := lipgloss.NewStyle().Foreground(style.Muted)

	var statusText string
	if serviceStats != nil {
		// 服務狀態
		var statusLine string
		if serviceStats.Status == "running" {
			statusLine = runningStyle.Render("● 運行中")
		} else {
			statusLine = stoppedStyle.Render("○ 已停止")
		}

		// 自啟動狀態
		var autoStartLine string
		if autoStart {
			autoStartLine = enabledStyle.Render("✓ 已啟用")
		} else {
			autoStartLine = disabledStyle.Render("✗ 已禁用")
		}

		// 構建信息塊
		lines := []string{}
		lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render(" 運行狀態:"), statusLine))

		if serviceStats.Status == "running" {
			lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render(" 運行時間:"),
				lipgloss.NewStyle().Foreground(style.Snow1).Render(serviceStats.Uptime)))
			lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render(" 內存佔用:"),
				lipgloss.NewStyle().Foreground(style.Snow1).Render(serviceStats.MemoryUsage)))
		}

		lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render(" 開機自啟:"), autoStartLine))

		statusText = strings.Join(lines, "\n")
	} else {
		statusText = lipgloss.NewStyle().
			Foreground(style.Muted).
			Render(" 正在加載服務狀態...")
	}

	// 根據服務狀態動態調整菜單
	var items []MenuItem
	if serviceStats != nil && serviceStats.Status == "running" {
		items = []MenuItem{
			{"", "", "", lipgloss.Color("")},
			{constants.KeyService_Restart, "重啟服務", "(重啟 sing-box 服務)", style.StatusYellow},
			{constants.KeyService_Stop, "停止服務", "(停止 sing-box 服務)", style.StatusRed},
			{constants.KeyService_Log, "查看實時日誌", "(tail -f 服務日誌)", style.Aurora2},
			{constants.KeyService_Refresh, "刷新狀態", "(重新獲取服務狀態)", style.Snow1},
			{constants.KeyService_AutoStart, "設置自啟動", "(開機自動啟動服務)", style.StatusGreen},
			{constants.KeyService_Health, "健康檢查", "(檢測服務運行狀況)", style.Snow1},
		}
	} else {
		items = []MenuItem{
			{"", "", "", lipgloss.Color("")},
			{constants.KeyService_Restart, "啟動服務", "(啟動 sing-box 服務)", style.StatusGreen},
			{constants.KeyService_Log, "查看實時日誌", "(查看報錯信息)", style.Aurora2},
			{constants.KeyService_Refresh, "刷新狀態", "(重新獲取服務狀態)", style.Snow1},
			{constants.KeyService_AutoStart, "設置自啟動", "(開機自動啟動服務)", style.StatusGreen},
			{constants.KeyService_Health, "健康檢查", "(檢測服務運行狀況)", style.Snow1},
		}
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

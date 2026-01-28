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

// RenderSwapMenu 渲染 Swap 菜單
func RenderSwapMenu(info *types.SwapInfo, inputMode bool, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("虛擬內存 (SWAP)")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 配置系統 SWAP 分區（弱雞 VPS 適用）")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)
	disabledStyle := lipgloss.NewStyle().Foreground(style.Muted)

	var statusText string
	if info != nil && info.Enabled {
		statusText = fmt.Sprintf(
			"%s %s\n%s %s\n%s %s\n%s %s\n%s %s",
			labelStyle.Render(" 狀態:"),
			valueStyle.Render("✓ 已啟用"),
			labelStyle.Render(" 總大小:"),
			valueStyle.Render(info.Total),
			labelStyle.Render(" 已使用:"),
			valueStyle.Render(info.Used),
			labelStyle.Render(" 剩餘:"),
			valueStyle.Render(info.Free),
			labelStyle.Render(" SWAP 文件:"),
			lipgloss.NewStyle().Foreground(style.Snow3).Render(info.SwapFile),
		)
	} else {
		statusText = fmt.Sprintf("%s %s",
			labelStyle.Render(" 狀態:"),
			disabledStyle.Render("✗ 未啟用"))
	}

	var mainContent string

	// 根據狀態決定顯示輸入框還是菜單
	if inputMode {
		prompt := lipgloss.NewStyle().Foreground(style.Aurora2).Bold(true).Render(" 請輸入 SWAP 大小 (單位 GB):")
		mainContent = fmt.Sprintf("\n%s\n%s", prompt, ti.View())
	} else {
		items := []MenuItem{
			{"", "", "", lipgloss.Color("")},
			{constants.KeySwap_Create, "創建 SWAP", "(自動創建 1GB SWAP 分區)", style.StatusGreen},
			{constants.KeySwap_Custom, "自定義大小", "(手動指定大小)", style.Snow1},
			{"", "", "", lipgloss.Color("")},
			{constants.KeySwap_Delete, "刪除 SWAP", "(移除現有分區)", style.StatusRed},
		}
		mainContent = renderMenuWithAlignment(items, 0, "", false)
	}

	statusBlock := RenderStatusMessage(statusMsg)
	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		statusText,
		mainContent,
		statusBlock,
		footer,
	)
}

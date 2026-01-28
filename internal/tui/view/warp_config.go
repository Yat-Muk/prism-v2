package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderWARPConfig 渲染 WARP 配置
func RenderWARPConfig(cfg *config.WARPConfig, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("WARP 出站管理")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 配置 Cloudflare WARP 作為出站代理")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 顯示當前狀態
	statusStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)

	var statusText string
	if cfg != nil && cfg.Enabled {
		statusText = fmt.Sprintf("%s %s",
			statusStyle.Render(" 狀態："),
			valueStyle.Render("✓ 已啓用"))
	} else {
		statusText = fmt.Sprintf("%s %s",
			statusStyle.Render(" 狀態："),
			lipgloss.NewStyle().Foreground(style.Muted).Render("✗ 未啓用"))
	}

	items := []MenuItem{
		{"", "", "", lipgloss.Color("")},
		{constants.KeyWARPConfig_Enable, "啓用 WARP", "(開啓 Cloudflare WARP 出站)", style.StatusGreen},
		{constants.KeyWARPConfig_Disable, "禁用 WARP", "(關閉 WARP 出站)", style.StatusRed},
		{constants.KeyWARPConfig_License, "配置許可證密鑰", "(設置 WARP+ 密鑰)", style.Snow1},
		{constants.KeyWARPConfig_Test, "測試連接", "(驗證 WARP 是否正常工作)", style.Aurora2},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 WARP 可用於解鎖 Cloudflare 保護的網站")

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

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

// RenderSNIProxyRouting 渲染 SNI 反向代理界面
func RenderSNIProxyRouting(cfg *config.SNIProxyConfig, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("SNI 反向代理")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 基於 SNI 的域名分流和反向代理")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 顯示當前狀態
	labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)
	disabledStyle := lipgloss.NewStyle().Foreground(style.Muted)

	var statusText string
	if cfg != nil && cfg.Enabled {
		domains := "無"
		if len(cfg.DomainRules) > 0 {
			domains = strings.Join(cfg.DomainRules, ", ")
		}

		statusText = fmt.Sprintf(
			"%s %s\n%s %s\n%s %s",
			labelStyle.Render(" 狀態："),
			valueStyle.Render("✓ 已啓用"),
			labelStyle.Render(" 目標 IP："),
			valueStyle.Render(cfg.TargetIP),
			labelStyle.Render(" 分流域名："),
			lipgloss.NewStyle().Foreground(style.Snow3).Render(domains),
		)
	} else {
		statusText = fmt.Sprintf("%s %s",
			labelStyle.Render(" 狀態："),
			disabledStyle.Render("✗ 未啓用"))
	}

	items := []MenuItem{
		{"", "", "", lipgloss.Color("")},
		{constants.KeyRouting_Enable, "啓用 SNI 代理", "(輸入目標 IP 地址)", style.StatusGreen},
		{constants.KeyRouting_Disable, "禁用 SNI 代理", "(關閉反向代理)", style.StatusRed},
		{constants.KeyRouting_AddDomain, "添加分流域名", "(指定域名走 SNI 代理)", style.Snow1},
		{constants.KeyRouting_Show, "查看配置", "(顯示完整配置)", style.Snow1},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 用於解鎖流媒體，需配合支持 SNI Proxy 的落地機")

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

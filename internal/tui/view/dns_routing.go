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

// RenderDNSRouting 渲染 DNS 分流界面
func RenderDNSRouting(cfg *config.DNSRoutingConfig, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("DNS 分流")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 自定義 DNS 服務器分流配置")

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
			labelStyle.Render(" DNS 服務器："),
			valueStyle.Render(cfg.Server),
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
		{constants.KeyRouting_Enable, "啓用 DNS 分流", "(輸入 DNS 服務器 IP)", style.StatusGreen},
		{constants.KeyRouting_Disable, "禁用 DNS 分流", "(關閉 DNS 分流)", style.StatusRed},
		{constants.KeyRouting_AddDomain, "添加分流域名", "(指定域名使用自定義 DNS)", style.Snow1},
		{constants.KeyRouting_Show, "查看配置", "(顯示完整 DNS 配置)", style.Snow1},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 用於解決 DNS 污染和流媒體解鎖")

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

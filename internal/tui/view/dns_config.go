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

// RenderDNSConfig 渲染 DNS 配置
func RenderDNSConfig(dnsConfig *config.DNSConfig, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("DNS 設置")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 配置 DNS 服務器和解析策略")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 顯示當前 DNS 服務器
	labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)

	var statusText string
	if dnsConfig != nil {
		enabledText := "已禁用"
		enabledColor := style.Muted
		if dnsConfig.Enabled {
			enabledText = "已启用"
			enabledColor = style.StatusGreen
		}

		statusText = fmt.Sprintf(
			"%s %s\n%s %s\n%s %s",
			labelStyle.Render(" 状态："),
			lipgloss.NewStyle().Foreground(enabledColor).Render(enabledText),
			labelStyle.Render(" DNS 服务器："),
			valueStyle.Render(strings.Join(dnsConfig.Servers, ", ")),
			labelStyle.Render(" 策略："),
			valueStyle.Render(dnsConfig.Strategy),
		)
	} else {
		statusText = lipgloss.NewStyle().
			Foreground(style.Muted).
			Render("DNS 未配置")
	}

	items := []MenuItem{
		{"", "", "", lipgloss.Color("")},
		{constants.KeyDNS_Toggle, "啓用/禁用 DNS", "(切換 DNS 功能)", style.Aurora3},
		{constants.KeyDNS_Servers, "修改 DNS 服務器", "(配置上游 DNS)", style.Snow1},
		{constants.KeyDNS_Strategy, "修改 DNS 策略", "(IPv4/IPv6 策略)", style.Snow1},
		{constants.KeyDNS_Rules, "配置 DNS 規則", "(域名分流規則)", style.Snow1},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 DNS 配置影響域名解析行為")

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

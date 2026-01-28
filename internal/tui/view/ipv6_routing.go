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

func RenderIPv6Routing(cfg *config.IPv6SplitConfig, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("IPv6 分流")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" IPv6 流量分流和路由策略")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)
	disabledStyle := lipgloss.NewStyle().Foreground(style.Muted)

	var statusText string
	if cfg != nil && cfg.Enabled {
		mode := "域名分流模式"
		if cfg.Global {
			mode = "全局模式"
		}

		domains := "無"
		if len(cfg.Domains) > 0 {
			domains = strings.Join(cfg.Domains, ", ")
		}

		statusText = fmt.Sprintf(
			"%s %s\n%s %s\n%s %s",
			labelStyle.Render(" 狀態："),
			valueStyle.Render("✓ 已啓用"),
			labelStyle.Render(" 模式："),
			valueStyle.Render(mode),
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
		{constants.KeyIPv6Split_Enable, "啓用 IPv6 分流", "(開啓 IPv6 流量路由)", style.StatusGreen},
		{constants.KeyIPv6Split_Disable, "禁用 IPv6 分流", "(關閉 IPv6 分流)", style.StatusRed},
		{constants.KeyIPv6Split_SetGlobal, "設置全局 IPv6", "(所有流量優先 IPv6)", style.StatusYellow},
		{constants.KeyIPv6Split_SetDomain, "添加分流域名", "(指定域名走 IPv6)", style.Snow1},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 適用於解鎖 Netflix 等限制 IPv4 的服務")

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

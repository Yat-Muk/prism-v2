package view

import (
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderIPv6Route 渲染 IPv6 路由配置
func RenderIPv6Route(enabled bool, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("IPv6 路由")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 配置 IPv6 流量分流和路由策略")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	items := []MenuItem{
		{constants.KeyIPv6Route_Enable, "啓用 IPv6 路由", "(開啓 IPv6 流量分流)", style.StatusGreen},
		{constants.KeyIPv6Route_Disable, "禁用 IPv6 路由", "(僅使用 IPv4)", style.StatusRed},
		{constants.KeyIPv6Route_Priority, "IPv6 優先級", "(高 / 中 / 低)", style.Snow1},
		{constants.KeyIPv6Route_DNS, "IPv6 DNS", "(配置 IPv6 DNS 服務器)", style.Snow1},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" ⚠️  需要服務器支持 IPv6")

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		menu,
		"",
		instruction,
		statusBlock,
		footer,
	)
}

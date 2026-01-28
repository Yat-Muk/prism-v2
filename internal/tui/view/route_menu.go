package view

import (
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderRouteMenu 渲染高級路由管理菜單
func RenderRouteMenu(cursor int, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("高級路由分流")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 服務端流量分流，用於解鎖 ChatGPT、流媒體等")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	items := []MenuItem{
		{constants.KeyRoute_WARP, "WARP 分流", "(WARP IPv4/IPv6 分流)", style.Snow1},
		{constants.KeyRoute_Socks5, "Socks5 分流", "(Socks5 入站/出站配置)", style.Snow1},
		{constants.KeyRoute_IPv6, "IPv6 分流", "(IPv6 流量分流)", style.Snow1},
		{constants.KeyRoute_DNS, "DNS 分流", "(自定義 DNS 服務器分流)", style.Snow1},
		{constants.KeyRoute_SNIProxy, "SNI 反向代理", "(SNI 反向代理分流)", style.Snow1},
	}

	menu := renderMenuWithAlignment(items, cursor, "", false)

	warning := lipgloss.NewStyle().
		Foreground(style.StatusYellow).
		Render(" ⚠️  分流功能需要 sing-box 支持")

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		menu,
		"",
		warning,
		statusBlock,
		footer,
	)
}

package view

import (
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderSNIProxy 渲染 SNI 反向代理配置
func RenderSNIProxy(ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("SNI 反向代理")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 基於 SNI 的域名分流和反向代理")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	items := []MenuItem{
		{constants.KeySNIProxy_Enable, "啟用 SNI 代理", "(開啓基於域名的分流)", style.StatusGreen},
		{constants.KeySNIProxy_Disable, "禁用 SNI 代理", "(關閉域名分流)", style.StatusRed},
		{constants.KeySNIProxy_Add, "添加 分流規則", "(指定域名走特定出站)", style.Snow1},
		{constants.KeySNIProxy_List, "查看 規則列表", "(顯示所有分流規則)", style.Snow1},
		{constants.KeySNIProxy_Delete, "刪除 分流規則", "(移除指定分流規則)", style.StatusYellow},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 可實現按域名分流到不同出站")

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

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

// RenderSocks5Routing 渲染 Socks5 分流菜單
func RenderSocks5Routing(cfg *config.Socks5Config, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("Socks5 分流")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 配置 Socks5 入站/出站分流")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 顯示當前狀態
	labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)
	disabledStyle := lipgloss.NewStyle().Foreground(style.Muted)

	var statusText string
	if cfg != nil {
		inboundStatus := "✗ 未啓用"
		if cfg.Inbound.Enabled {
			inboundStatus = fmt.Sprintf("✓ 已啓用 (端口: %d)", cfg.Inbound.Port)
		}

		outboundStatus := "✗ 未啓用"
		if cfg.Outbound.Enabled {
			outboundStatus = fmt.Sprintf("✓ 已啓用 (%s:%d)", cfg.Outbound.Server, cfg.Outbound.Port)
		}

		statusText = fmt.Sprintf(
			"%s %s\n%s %s",
			labelStyle.Render(" 入站狀態："),
			valueStyle.Render(inboundStatus),
			labelStyle.Render(" 出站狀態："),
			valueStyle.Render(outboundStatus),
		)
	} else {
		statusText = fmt.Sprintf("%s %s",
			labelStyle.Render(" 狀態："),
			disabledStyle.Render("未知"))
	}

	items := []MenuItem{
		{"", "", "", lipgloss.Color("")},
		{constants.KeySocks5_Inbound, "Socks5 入站", "(配置入站: 解鎖機、落地機)", style.Snow1},
		{constants.KeySocks5_Outbound, "Socks5 出站", "(配置出站: 轉發機、代理機)", style.Snow1},
		{constants.KeySocks5_ShowConfig, "查看配置", "(顯示當前 Socks5 配置)", style.Aurora2},
		{"", "", "", lipgloss.Color("")},
		{constants.KeySocks5_Uninstall, "卸載", "(移除 Socks5 分流)", style.StatusRed},
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

// RenderSocks5Inbound 渲染 Socks5 入站配置
func RenderSocks5Inbound(cfg *config.Socks5InboundConfig, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("Socks5 入站配置")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 配置本機作為 Socks5 代理服務器")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	items := []MenuItem{
		{constants.KeySocks5In_Toggle, "啓用/禁用", "(開關 Socks5 入站)", style.Aurora1},
		{constants.KeySocks5In_Port, "設置端口", fmt.Sprintf("(當前: %d)", cfg.Port), style.Snow1},
		{constants.KeySocks5In_Auth, "設置認證", "(配置用戶名密碼)", style.Snow1},
		{constants.KeySocks5In_IP, "允許的 IP", "(配置允許訪問的 IP 地址)", style.Snow1},
		{constants.KeySocks5In_Rule, "分流規則", "(配置域名分流規則)", style.Aurora2},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 此端口需要配置到其他機器出站")

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

// RenderSocks5Outbound 渲染 Socks5 出站配置
func RenderSocks5Outbound(cfg *config.Socks5OutboundConfig, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("Socks5 出站配置")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 配置轉發機、代理機的 Socks5 出站")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	items := []MenuItem{
		{constants.KeySocks5Out_Toggle, "啓用/禁用", "(開關 Socks5 出站)", style.Aurora1},
		{constants.KeySocks5Out_Server, "修改 落地機地址", fmt.Sprintf("(當前: %s:%d)", cfg.Server, cfg.Port), style.Snow1},
		{constants.KeySocks5Out_Auth, "設置認證", "(配置用戶名密碼)", style.Snow1},
		{constants.KeySocks5Out_Global, "全局轉發", "(所有流量通過 Socks5))", style.StatusYellow},
		{constants.KeySocks5Out_Rule, "分流規則", "(指定域名分流)", style.Aurora2},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		menu,
		statusBlock,
		footer,
	)
}

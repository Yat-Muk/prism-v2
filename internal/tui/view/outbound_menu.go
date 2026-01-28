package view

import (
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderOutboundMenu 渲染出口策略菜單（根據 IP 支持情況動態調整）
func RenderOutboundMenu(cursor int, currentStrategy string, ti textinput.Model, hasIPv4, hasIPv6 bool, statusMsg string) string {
	items := []MenuItem{}

	// 根據支持情況添加選項
	if hasIPv4 && hasIPv6 {
		// 雙棧支持 - 顯示所有選項
		items = append(items,
			MenuItem{"", "", "", lipgloss.Color("")},
			MenuItem{constants.KeyOutbound_PreferIPv4, "IPv4 優先", "(prefer_ipv4) [推薦]", style.Snow1},
			MenuItem{constants.KeyOutbound_PreferIPv6, "IPv6 優先", "(prefer_ipv6)", style.Snow1},
			MenuItem{constants.KeyOutbound_IPv4Only, "僅 IPv4", "(ipv4_only)", style.Snow1},
			MenuItem{constants.KeyOutbound_IPv6Only, "僅 IPv6", "(ipv6_only)", style.Snow1},
		)
	} else if hasIPv4 {
		// 僅支持 IPv4
		items = append(items,
			MenuItem{"", "", "", lipgloss.Color("")},
			MenuItem{constants.KeyOutbound_PreferIPv4, "僅 IPv4", "(ipv4_only) ✓ 當前唯一可用", style.Aurora1},
			MenuItem{constants.KeyOutbound_PreferIPv6, "IPv6 優先", "(prefer_ipv6) ✗ 不可用", style.Muted},
		)
	} else if hasIPv6 {
		// 僅支持 IPv6
		items = append(items,
			MenuItem{"", "", "", lipgloss.Color("")},
			MenuItem{constants.KeyOutbound_IPv6Only, "僅 IPv6", "(ipv6_only) ✓ 當前唯一可用", style.Aurora1},
			MenuItem{constants.KeyOutbound_PreferIPv4, "IPv4 優先", "(prefer_ipv4) ✗ 不可用", style.Muted},
		)
	} else {
		// 無網絡??
		items = append(items,
			MenuItem{"", "", "", lipgloss.Color("")},
			MenuItem{"", "檢測到網絡異常", "", style.StatusYellow},
			MenuItem{"", "請檢查網絡配置", "", style.Snow3},
		)
	}

	header := renderSubpageHeader("出站策略")

	// 策略名稱映射邏輯
	displayStrategy := currentStrategy
	switch currentStrategy {
	case "prefer_ipv4":
		displayStrategy = "IPv4 優先"
	case "prefer_ipv6":
		displayStrategy = "IPv6 優先"
	case "ipv4_only":
		displayStrategy = "僅 IPv4"
	case "ipv6_only":
		displayStrategy = "僅 IPv6"
	case "":
		displayStrategy = "默認 (IPv4 優先)"
	}

	// 當前策略 + IP 支持狀態
	statusLines := []string{}
	if currentStrategy != "" {
		labelStyle := lipgloss.NewStyle().Foreground(style.Snow2)
		nameStyle := lipgloss.NewStyle().Foreground(style.Aurora4)
		statusLines = append(statusLines, lipgloss.JoinHorizontal(
			lipgloss.Left,
			labelStyle.Render(" 當前策略: "),
			nameStyle.Render(displayStrategy),
		))
	}

	infoSep := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// IP 支持狀態
	ipStatus := lipgloss.NewStyle().Foreground(style.Snow2).Render(" IP 支持: ")
	if hasIPv4 && hasIPv6 {
		ipStatus += lipgloss.NewStyle().Foreground(style.StatusYellow).Render("IPv4 ✓  IPv6 ✓")
	} else if hasIPv4 {
		ipStatus += lipgloss.NewStyle().Foreground(style.StatusYellow).Render("IPv4 ✓  ") +
			lipgloss.NewStyle().Foreground(style.Snow3).Render("IPv6 ✗")
	} else if hasIPv6 {
		ipStatus += lipgloss.NewStyle().Foreground(style.Snow3).Render("IPv4 ✗  ") +
			lipgloss.NewStyle().Foreground(style.StatusYellow).Render("IPv6 ✓")
	} else {
		ipStatus += lipgloss.NewStyle().Foreground(style.StatusRed).Render("未檢測到有效 IP")
	}

	infoBlock := lipgloss.JoinVertical(
		lipgloss.Left,
		statusLines...,
	)

	infoBlock = lipgloss.JoinVertical(lipgloss.Left, infoBlock, infoSep, ipStatus)

	menu := renderMenuWithAlignment(items, cursor, "", false)

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		infoBlock,
		menu,
		statusBlock,
		footer,
	)
}

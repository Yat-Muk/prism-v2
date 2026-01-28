package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/Yat-Muk/prism-v2/internal/tui/types"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderStreamingCheck 渲染流媒體檢測界面
func RenderStreamingCheck(result *types.StreamingCheckResult, checking bool, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("流媒體/IP 檢測")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 實時檢測服務器 IP 與流媒體解鎖狀態")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	var content string

	if checking {
		// --- 1. 檢測中狀態 ---
		spinner := lipgloss.NewStyle().Foreground(style.Aurora2).Blink(true).Render("⣾⣽⣻⢿⡿⣟⣯⣷")
		content = fmt.Sprintf("\n\n      %s  正在深度檢測中...\n \n                請稍候 10-20 秒...\n\n", spinner)
	} else if result != nil {
		// --- 2. 檢測完成顯示結果 ---

		labelStyle := lipgloss.NewStyle().Foreground(style.Snow3).Width(12)
		valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)

		sectionStyle := lipgloss.NewStyle().
			Foreground(style.Snow2).
			PaddingTop(1).
			PaddingBottom(1).
			Render("--- 流媒體解鎖詳情 ---")

		content = fmt.Sprintf(
			" %s %s\n %s %s\n %s %s\n %s\n"+
				" %s %s\n"+
				" %s %s\n"+
				" %s %s\n"+
				" %s %s\n"+
				" %s %s",
			labelStyle.Render("IPv4 地址:"), valueStyle.Render(result.IPv4),
			labelStyle.Render("IPv6 地址:"), valueStyle.Render(result.IPv6),
			labelStyle.Render("IP 地理位置:"), valueStyle.Render(result.Location),
			sectionStyle,
			labelStyle.Render("Netflix:"), formatStatus(result.Netflix),
			labelStyle.Render("ChatGPT:"), formatStatus(result.ChatGPT),
			labelStyle.Render("Disney+:"), formatStatus(result.Disney),
			labelStyle.Render("YouTube:"), formatStatus(result.YouTube),
			labelStyle.Render("TikTok:"), formatStatus(result.TikTok),
		)
	}

	bottomSeparator := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(strings.Repeat("═", 50))

	navigation := lipgloss.NewStyle().
		Foreground(style.Muted).
		PaddingTop(1).
		Render(" Esc 返回 • Enter 重新檢測")

	statusBlock := RenderStatusMessage(statusMsg)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		content,
		bottomSeparator,
		statusBlock,
		navigation,
	)
}

// 根據字符符號渲染顏色
func formatStatus(status string) string {
	// 定義樣式
	green := lipgloss.NewStyle().Foreground(style.StatusGreen).Bold(true)
	yellow := lipgloss.NewStyle().Foreground(style.StatusYellow).Bold(true)
	red := lipgloss.NewStyle().Foreground(style.StatusRed).Bold(true)
	gray := lipgloss.NewStyle().Foreground(style.Snow3)

	// 特殊處理 ChatGPT 的複合狀態 "Web: ? / App: ✓"
	if strings.Contains(status, "Web:") {
		parts := strings.Split(status, "/")
		var res []string
		for _, p := range parts {
			if strings.Contains(p, "✓") {
				res = append(res, strings.ReplaceAll(p, "✓", green.Render("✓")))
			} else if strings.Contains(p, "?") {
				// 問號顯示為黃色，表示 WAF 攔截不確定
				res = append(res, strings.ReplaceAll(p, "?", yellow.Render("? (CF擋)")))
			} else {
				res = append(res, strings.ReplaceAll(p, "✗", red.Render("✗")))
			}
		}
		return strings.Join(res, " / ")
	}

	// 其他狀態處理 (✓, !, ✗)
	if strings.Contains(status, "✓") {
		text := strings.ReplaceAll(status, "✓", "")
		return green.Render("✓") + text
	}
	if strings.Contains(status, "!") {
		text := strings.ReplaceAll(status, "!", "")
		return yellow.Render("!") + text
	}
	if strings.Contains(status, "✗") {
		text := strings.ReplaceAll(status, "✗", "")
		return red.Render("✗") + text
	}

	return gray.Render(status)
}

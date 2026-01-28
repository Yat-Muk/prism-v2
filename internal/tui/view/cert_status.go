package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/Yat-Muk/prism-v2/internal/tui/types"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderCertStatus 渲染證書狀態界面
func RenderCertStatus(certList []types.CertInfo, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("證書狀態")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 查看所有證書的狀態和有效期")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 空列表處理
	if len(certList) == 0 {
		emptyBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(style.Polar4).
			Padding(1, 2).
			Foreground(style.Snow3).
			Render("暫無證書\n\n• 輸入 0 返回上級菜單\n• 可申請 ACME 證書")
		return lipgloss.JoinVertical(lipgloss.Left, header, desc, divider, "\n"+emptyBox, "\n")
	}

	// 表頭
	tableHeader := " " + lipgloss.NewStyle().
		Foreground(style.Aurora4).
		Bold(true).
		Render(fmt.Sprintf("%-34s  %-26s  %-20s  %s", "域名", "過期時間", "剩餘", "狀態"))

	var rows []string
	rows = append(rows, tableHeader)
	rows = append(rows, lipgloss.NewStyle().Foreground(style.Polar4).Render(strings.Repeat("-", 95)))

	seen := make(map[string]bool)

	for _, cert := range certList {
		// 去重顯示 (防止同域名多個證書文件刷屏)
		if seen[cert.Domain] {
			continue
		}
		seen[cert.Domain] = true

		// 狀態樣式處理
		// 我們直接使用 types.CertInfo 中已經計算好的 DaysLeft 和 Status
		var statusStyle lipgloss.Style
		var statusText string

		switch cert.Status {
		case "Expired":
			statusStyle = lipgloss.NewStyle().Foreground(style.StatusRed).Bold(true)
			statusText = "✗ 過期"
		case "Expiring": // < 30 天
			statusStyle = lipgloss.NewStyle().Foreground(style.StatusYellow).Bold(true)
			statusText = "⚠ 警告"
		case "Valid":
			// 如果剩餘天數極少但未標記為 Expiring (防止邏輯漏網)
			if cert.DaysLeft <= 7 {
				statusStyle = lipgloss.NewStyle().Foreground(style.StatusRed).Bold(true)
				statusText = "⚠ 危險"
			} else {
				statusStyle = lipgloss.NewStyle().Foreground(style.StatusGreen)
				statusText = "✓ 正常"
			}
		default:
			statusStyle = lipgloss.NewStyle().Foreground(style.Muted)
			statusText = cert.Status
		}

		// 域名截斷
		domain := cert.Domain
		if len(domain) > 32 {
			domain = domain[:29] + "..."
		}

		daysStr := fmt.Sprintf("%d天", cert.DaysLeft)

		// 格式化行
		row := " " + fmt.Sprintf("%-34s  %-26s  %-20s  %s",
			lipgloss.NewStyle().Foreground(style.Aurora1).Render(domain),
			lipgloss.NewStyle().Foreground(style.Snow2).Render(cert.ExpireDate),
			lipgloss.NewStyle().Foreground(style.Snow3).Render(daysStr),
			statusStyle.Render(statusText),
		)
		rows = append(rows, row)
	}

	// 拼接列表
	listContent := strings.Join(rows, "\n")

	// 底部說明
	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render("\n 說明：\n  • 系統每日自動檢查並續期剩餘 30 天內的證書\n  • 按 [R] 鍵可手動刷新列表")

	statusBlock := RenderStatusMessage(statusMsg)
	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		"\n"+listContent,
		instruction,
		statusBlock,
		footer,
	)
}

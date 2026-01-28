package view

import (
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderCertMenu 渲染證書管理菜單
func RenderCertMenu(acmeDomains, selfSignedDomains []string, cursor int, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("證書管理")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 管理 ACME 證書申請、自簽證書生成及狀態監控")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 證書概況行
	labelStyle := lipgloss.NewStyle().Foreground(style.Snow2)
	valueStyle := lipgloss.NewStyle().Foreground(style.Aurora4)

	acmeLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		labelStyle.Render(" ACME 證書: "),
		valueStyle.Render(formatDomains(acmeDomains)),
	)

	selfLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		labelStyle.Render(" 自簽名證書: "),
		valueStyle.Render(formatDomains(selfSignedDomains)),
	)

	items := []MenuItem{
		{"", "", "", lipgloss.Color("")},
		{constants.KeyCert_ApplyHTTP, "申請 ACME 證書", "(80端口模式 - 需釋放端口)", style.Aurora1},
		{constants.KeyCert_ApplyDNS, "申請 ACME 證書", "(DNS API模式)", style.Snow1},
		{constants.KeyCert_SwitchProvider, "切換證書提供商", "(Let’s Encrypt / ZeroSSL)", style.Snow1},
		{constants.KeyCert_Renew, "續期現有證書", "(手動觸發續期任務)", style.Snow1},
		{constants.KeyCert_Status, "查看證書狀態", "(到期時間/頒發者)", style.Snow1},
		{"", "", "", lipgloss.Color("")},
		{constants.KeyCert_Delete, "刪除域名證書", "(僅在重新配置時使用，請謹慎)", style.StatusRed},
		{"", "", "", lipgloss.Color("")},
		{constants.KeyCert_ModeSwitch, "切換證書模式", "(獨立設置協議證書)", style.Aurora4},
	}

	menu := renderMenuWithAlignment(items, cursor, "", false)

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		acmeLine,
		selfLine,
		menu,
		statusBlock,
		footer,
	)
}

func formatDomains(domains []string) string {
	if len(domains) == 0 {
		return "無"
	}
	if len(domains) <= 3 {
		return strings.Join(domains, ", ")
	}
	return strings.Join(domains[:3], ", ") + " ..."
}

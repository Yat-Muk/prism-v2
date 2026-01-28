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

// RenderWARPRouting 渲染 WARP 分流界面
func RenderWARPRouting(cfg *config.WARPConfig, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("WARP 分流")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" Cloudflare WARP 流量分流配置")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 顯示當前狀態
	labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)
	disabledStyle := lipgloss.NewStyle().Foreground(style.Muted)
	warnStyle := lipgloss.NewStyle().Foreground(style.StatusYellow)

	var statusText string
	var isGlobal bool // 用於判斷菜單顯示

	if cfg != nil && cfg.Enabled {
		isGlobal = cfg.Global

		modeText := "域名分流 (Split)"
		if cfg.Global {
			modeText = warnStyle.Render("全局代理 (Global)")
		} else {
			modeText = valueStyle.Render(modeText)
		}

		// [修復 3] 使用截斷函數處理長域名
		domains := formatDomainList(cfg.Domains)

		statusText = fmt.Sprintf(
			"%s %s\n%s %s\n%s %s",
			labelStyle.Render(" 狀態："),
			valueStyle.Render("✓ 已啓用 (雙棧隧道)"), // WARP 通常是雙棧的
			labelStyle.Render(" 模式："),
			modeText,
			labelStyle.Render(" 分流域名："),
			lipgloss.NewStyle().Foreground(style.Snow3).Render(domains),
		)
	} else {
		statusText = fmt.Sprintf("%s %s",
			labelStyle.Render(" 狀態："),
			disabledStyle.Render("✗ 未啓用"))
	}

	// [修復 2] 動態改變菜單文字
	globalSwitchText := "切換為 全局模式"
	globalSwitchDesc := "(所有流量走 WARP)"
	if isGlobal {
		globalSwitchText = "切換為 分流模式"
		globalSwitchDesc = "(僅指定域名走 WARP)"
	}

	items := []MenuItem{
		{"", "", "", lipgloss.Color("")},
		{constants.KeyWARP_ToggleIPv4, "啓用 WARP", "(開啓 WARP 隧道)", style.StatusGreen},
		{constants.KeyWARP_ToggleIPv6, "重啟 WARP", "(重新連接隧道)", style.Snow1},
		{constants.KeyWARP_SetGlobal, globalSwitchText, globalSwitchDesc, style.StatusYellow},
		{constants.KeyWARP_SetDomains, "添加分流域名", "(指定域名走 WARP)", style.Snow1},
		{constants.KeyWARP_ShowConfig, "查看配置", "(顯示完整 WARP 配置)", style.Snow1},
		{constants.KeyWARP_Disable, "禁用 WARP", "(關閉 WARP 分流)", style.StatusRed},
		{"", "", "", lipgloss.Color("")},
		{constants.KeyWARP_SetLicense, "配置許可證", "(WARP+ 密鑰 / 留空免費版)", style.Snow1},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	warning := lipgloss.NewStyle().
		Foreground(style.StatusYellow).
		Render(" ⚠️  需要配置 WARP 密鑰才能使用")

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 用於解鎖 ChatGPT、流媒體等服務")

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
		warning,
		"",
		instruction,
		statusBlock,
		footer,
	)
}

// 域名列表截斷輔助函數
func formatDomainList(domains []string) string {
	if len(domains) == 0 {
		return "無"
	}

	// 總字符長度限制
	const maxChars = 50
	// 顯示數量限制
	const maxCount = 3

	var display []string
	currentLen := 0

	for i, d := range domains {
		// 如果超過數量限制或字符限制
		if i >= maxCount || currentLen+len(d) > maxChars {
			remaining := len(domains) - i
			return strings.Join(display, ", ") + fmt.Sprintf(", ... (還有 %d 個)", remaining)
		}
		display = append(display, d)
		currentLen += len(d) + 2 // +2 for ", "
	}

	return strings.Join(display, ", ")
}

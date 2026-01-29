package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/Yat-Muk/prism-v2/internal/tui/types"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderMainView 渲染主視圖
func RenderMainView(
	stats *types.SystemStats,
	serviceStats *types.ServiceStats,
	cursor int,
	width int,
	isInstalled bool,
	coreVersion string,
	hasUpdate bool,
	latestVersion string,
	scriptVersion string,
	ti textinput.Model,
	statusMsg string,
) string {
	if width < 80 {
		width = 80
	}

	var sections []string

	// 1. Logo 和標題
	sections = append(sections, renderHeader(scriptVersion))

	// 2. 系統信息面板
	sections = append(sections, renderSystemInfoPanel(stats, serviceStats, width, isInstalled, coreVersion, hasUpdate, latestVersion))

	// 3. 菜單選項
	sections = append(sections, renderMainMenu(-1, isInstalled))

	// 4. 插入狀態框
	statusBlock := RenderStatusMessage(statusMsg)

	// 5. 底部提示
	promptLine := RenderTextInput(ti)

	// 6. 組合整體佈局
	mainContent := lipgloss.JoinVertical(
		lipgloss.Left,
		strings.Join(sections, "\n"),
		statusBlock,
		promptLine,
	)

	return mainContent
}

// renderHeader 渲染頭部 (保持不變)
func renderHeader(scriptVersion string) string {
	logo := RenderLogo()

	labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	valueStyle := lipgloss.NewStyle().Foreground(style.Snow2)

	subtitle := lipgloss.NewStyle().
		Foreground(style.Aurora3).
		Width(50).
		AlignHorizontal(lipgloss.Center).
		Render(":: 現代化 sing-box 管理工具 ::")

	versionText := scriptVersion
	if versionText == "" {
		versionText = "檢查中..."
	} else if !strings.HasPrefix(versionText, "v") {
		versionText = "v" + versionText
	}

	infoContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		labelStyle.Render("腳本版本: "),
		valueStyle.Render(versionText),
	)

	info := lipgloss.NewStyle().
		Width(49).
		AlignHorizontal(lipgloss.Center).
		Render(infoContent)

	projectContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		labelStyle.Render("項目地址: "),
		valueStyle.Render("https://github.com/Yat-Muk/prism-v2"),
	)

	projectURL := lipgloss.NewStyle().
		Width(49).
		AlignHorizontal(lipgloss.Center).
		Render(projectContent)

	separator := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(strings.Repeat("═", 50))

	return lipgloss.JoinVertical(lipgloss.Left,
		logo,
		"",
		subtitle,
		"",
		info,
		projectURL,
		separator,
	)
}

// renderSystemInfoPanel 渲染系統信息面板
func renderSystemInfoPanel(
	stats *types.SystemStats,
	serviceStats *types.ServiceStats,
	width int,
	isInstalled bool,
	coreVersion string,
	hasUpdate bool,
	latestVersion string,
) string {
	labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	valueStyle := lipgloss.NewStyle().Foreground(style.StatusGreen)
	ipValueStyle := lipgloss.NewStyle().Foreground(style.Snow2)
	valueMuted := lipgloss.NewStyle().Foreground(style.Muted)
	greenArrow := func(dir string) string {
		return lipgloss.NewStyle().Foreground(style.StatusGreen).Render(dir)
	}

	var lines []string

	if stats == nil {
		line1 := lipgloss.JoinHorizontal(
			lipgloss.Left,
			labelStyle.Render("系統: "),
			valueMuted.Render(fmt.Sprintf("%-18s", "檢查中...")),
			labelStyle.Render("內核: "),
			valueMuted.Render(fmt.Sprintf("%-10s", "檢查中...")),
			labelStyle.Render("BBR: "),
			valueMuted.Render("檢查中..."),
		)
		lines = append(lines, line1)

		line2 := fmt.Sprintf(
			"%s%s%s%s%s%s",
			labelStyle.Render("資源: CPU: "),
			valueMuted.Render("檢查中..."),
			labelStyle.Render("內存: "),
			valueMuted.Render("檢查中..."),
			labelStyle.Render("磁盤: "),
			valueMuted.Render("檢查中..."),
		)
		lines = append(lines, line2)

		line3 := fmt.Sprintf(
			"%s%s%s%s%s%s",
			labelStyle.Render("網絡: 上傳"),
			greenArrow("↑ "),
			valueMuted.Render("檢查中..."),
			labelStyle.Render("下載"),
			greenArrow("↓ "),
			valueMuted.Render("檢查中..."),
		)
		lines = append(lines, line3)

		separator := lipgloss.NewStyle().Foreground(style.Snow2).Render(strings.Repeat("─", 50))
		lines = append(lines, separator)

		lines = append(lines,
			fmt.Sprintf("%s%s", labelStyle.Render("核心版本: "), valueMuted.Render("檢查中...")),
			fmt.Sprintf("%s%s", labelStyle.Render("運行狀態: "), valueMuted.Render("檢查中...")),
			fmt.Sprintf("%s%s", labelStyle.Render("出口策略: "), valueMuted.Render("檢查中...")),
			fmt.Sprintf("%s%s", labelStyle.Render("IPv4地址: "), valueMuted.Render("檢查中...")),
			fmt.Sprintf("%s%s", labelStyle.Render("IPv6地址: "), valueMuted.Render("檢查中...")),
		)

		separator2 := lipgloss.NewStyle().Foreground(style.Snow2).Render(strings.Repeat("═", 50))
		return lipgloss.JoinVertical(lipgloss.Left, lipgloss.JoinVertical(lipgloss.Left, lines...), separator2)
	}

	osDisplay := stats.OS
	if osDisplay == "" {
		osDisplay = "Linux"
	} else {
		osDisplay = strings.ReplaceAll(osDisplay, "LTS", "")
		osDisplay = strings.TrimSpace(osDisplay)
	}

	r := []rune(osDisplay)
	if len(r) > 18 {
		osDisplay = string(r[:16]) + ".."
	}

	kernelVersion := stats.Kernel
	if parts := strings.Split(kernelVersion, "-"); len(parts) > 0 {
		kernelVersion = parts[0]
	}
	if len(kernelVersion) > 10 {
		kernelVersion = kernelVersion[:10]
	}

	line1 := lipgloss.JoinHorizontal(
		lipgloss.Left,
		labelStyle.Render("系統: "),
		ipValueStyle.Render(fmt.Sprintf("%-18s", osDisplay)),
		labelStyle.Render("內核: "),
		ipValueStyle.Render(fmt.Sprintf("%-9s", kernelVersion)),
		labelStyle.Render("BBR: "),
		ipValueStyle.Render(stats.BBR),
	)
	lines = append(lines, line1)

	line2 := fmt.Sprintf(
		"%s%s%s%s%s%s",
		labelStyle.Render("資源: CPU: "),
		valueStyle.Render(fmt.Sprintf("%-8s", percentString(stats.CPUUsage))),
		labelStyle.Render("內存: "),
		valueStyle.Render(fmt.Sprintf("%-9s", percentString(stats.MemUsage))),
		labelStyle.Render("磁盤: "),
		valueStyle.Render(percentString(stats.DiskUsage)),
	)
	lines = append(lines, line2)

	line3 := fmt.Sprintf(
		"%s%s%s   %s%s%s",
		labelStyle.Render("網絡: 上傳"),
		greenArrow("↑ "),
		valueStyle.Render(stats.NetSentRate),
		labelStyle.Render("下載"),
		greenArrow("↓ "),
		valueStyle.Render(stats.NetRecvRate),
	)
	lines = append(lines, line3)

	separator := lipgloss.NewStyle().Foreground(style.Snow2).Render(strings.Repeat("─", 50))
	lines = append(lines, separator)

	var coreVersionLine string
	switch {
	case !isInstalled:
		coreVersionLine = fmt.Sprintf("%s%s", labelStyle.Render("核心版本: "), lipgloss.NewStyle().Foreground(style.Snow2).Render("未安裝"))
	case coreVersion == "":
		coreVersionLine = fmt.Sprintf("%s%s", labelStyle.Render("核心版本: "), lipgloss.NewStyle().Foreground(style.Snow2).Render("檢查中..."))
	case hasUpdate && latestVersion != "":
		coreVersionLine = fmt.Sprintf("%s%s %s",
			labelStyle.Render("核心版本: "),
			lipgloss.NewStyle().Foreground(style.Aurora2).Render("sing-box "+coreVersion),
			lipgloss.NewStyle().Foreground(style.StatusYellow).Render("(可更新至 "+latestVersion+")"),
		)
	default:
		coreVersionLine = fmt.Sprintf("%s%s", labelStyle.Render("核心版本: "), lipgloss.NewStyle().Foreground(style.Aurora2).Render("sing-box "+coreVersion))
	}
	lines = append(lines, coreVersionLine)

	line5 := renderEnhancedServiceStatusWithStyles(serviceStats, labelStyle, valueStyle)
	lines = append(lines, line5)

	line6 := fmt.Sprintf("%s%s", labelStyle.Render("出口策略: "), lipgloss.NewStyle().Foreground(style.StatusYellow).Render("IPv4 優先"))
	lines = append(lines, line6)

	ipv4 := stats.IPv4
	if ipv4 == "" {
		ipv4 = "N/A"
	}
	ipv6 := stats.IPv6
	if ipv6 == "" {
		ipv6 = "N/A"
	}

	lines = append(lines,
		fmt.Sprintf("%s%s", labelStyle.Render("IPv4地址: "), ipValueStyle.Render(ipv4)),
		fmt.Sprintf("%s%s", labelStyle.Render("IPv6地址: "), ipValueStyle.Render(ipv6)),
	)

	separator2 := lipgloss.NewStyle().Foreground(style.Snow2).Render(strings.Repeat("═", 50))
	return lipgloss.JoinVertical(lipgloss.Left, lipgloss.JoinVertical(lipgloss.Left, lines...), separator2)
}

func percentString(p float64) string {
	return fmt.Sprintf("%.1f%%", p)
}

// renderEnhancedServiceStatusWithStyles 渲染服務狀態
func renderEnhancedServiceStatusWithStyles(
	serviceStats *types.ServiceStats,
	labelStyle lipgloss.Style,
	valueStyle lipgloss.Style,
) string {
	unknownStyle := lipgloss.NewStyle().Foreground(style.Muted)

	if serviceStats == nil {
		return labelStyle.Render("運行狀態: ") +
			unknownStyle.Render("○ 已停止")
	}

	icon := "○"
	stateLabel := "已停止"
	stateColor := style.StatusRed

	switch serviceStats.Status {
	case "active", "running":
		icon = "●"
		stateLabel = "運行中"
		stateColor = style.StatusGreen
	case "activating":
		icon = "◐"
		stateLabel = "啟動中"
		stateColor = style.StatusYellow
	case "failed":
		icon = "✗"
		stateLabel = "異常"
		stateColor = style.StatusOrange
	}

	statusPart := lipgloss.NewStyle().
		Foreground(stateColor).
		Render(fmt.Sprintf("%s %s", icon, stateLabel))

	if serviceStats.Status == "active" || serviceStats.Status == "running" {
		var uptimePart string
		if serviceStats.Uptime != "" {
			uptimePart = valueStyle.Render(serviceStats.Uptime)
		} else {
			uptimePart = unknownStyle.Render("未知")
		}

		var memPart string
		if serviceStats.MemoryUsage != "" {
			memPart = valueStyle.Render(serviceStats.MemoryUsage)
		} else {
			memPart = unknownStyle.Render("未知")
		}

		return fmt.Sprintf(
			"%s%s %s%s %s%s",
			labelStyle.Render("運行狀態: "),
			statusPart,
			labelStyle.Render("時間: "),
			uptimePart,
			labelStyle.Render("內存: "),
			memPart,
		)
	}

	return fmt.Sprintf(
		"%s%s",
		labelStyle.Render("運行狀態: "),
		statusPart,
	)
}

func renderMainMenu(cursor int, isInstalled bool) string {
	item1Text := "安裝部署 Prism"
	item1Color := style.StatusGreen
	if isInstalled {
		item1Text = "重新部署 Prism"
		item1Color = style.StatusRed
	}

	items := []MenuItem{
		{constants.KeyMain_InstallWizard, item1Text, "", item1Color},
		{constants.KeyMain_ServiceStart, "啟動/重啟 服務", "", style.Snow1},
		{constants.KeyMain_ServiceStop, "停止 服務", "", style.Snow1},

		{"", "", "", lipgloss.Color("")},

		{constants.KeyMain_Config, "配置與協議", "(協議開關/配置/SNI 域名/UUID/端口)", style.Snow1},
		{constants.KeyMain_Cert, "證書管理", "(ACME 證書申請/證書切換)", style.Snow1},
		{constants.KeyMain_Outbound, "出口策略", "(切換 IPv4/IPv6 優先級)", style.Snow1},
		{constants.KeyMain_Route, "路由與分流", "(WARP/Socks5/IPv6/DNS/SNI 反向代理)", style.Snow1},

		{"", "", "", lipgloss.Color("")},

		{constants.KeyMain_Core, "核心與更新", "(核心/腳本版本管理)", style.Snow1},
		{constants.KeyMain_Tools, "實用工具箱", "(BBR/Swap/SSH 防護/IP 檢測/備份)", style.Snow1},
		{constants.KeyMain_Log, "日誌管理", "(實時日誌/日誌級別切換/導出)", style.Snow1},
		{constants.KeyMain_NodeInfo, "節點信息", "(鏈接/二維碼/客戶端 JSON)", style.StatusGreen},

		{"", "", "", lipgloss.Color("")},

		{constants.KeyMain_Uninstall, "卸載 Prism", "(刪除程序和數據)", style.StatusRed},
		{constants.KeyMain_Quit, "退出", "", style.Snow1},
	}

	return renderMenuWithAlignment(items, cursor, "", false)
}

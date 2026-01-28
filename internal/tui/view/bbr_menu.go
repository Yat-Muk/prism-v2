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

// RenderBBRMenu 渲染 BBR 加速菜單
func RenderBBRMenu(info *types.BBRInfo, confirmMode bool, target string, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("BBR 加速與優化")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	var bodyContent string

	// [核心邏輯] 根據是否處於確認模式，渲染不同的內容
	if confirmMode {
		// ===========================
		// 模式 A: 安裝確認模式 (危險警告)
		// ===========================

		warnHeader := lipgloss.NewStyle().
			Foreground(style.StatusRed).
			Render("!!! 危險操作確認 !!!")

		// 根據安裝目標生成不同的警告文案
		var warnText string
		if target == "bbr2" {
			warnText = `
檢測到當前內核不支持 BBR2 算法。

系統將安裝 "BBR2/BBRplus 專用內核" 以支持該功能
1. 僅支持 x86_64 (amd64) 架構
2. 此操作將替換現有內核，存在導致無法啟動的風險
3. 強烈建議先備份數據`
		} else {
			warnText = `
即將安裝 XanMod 高性能內核 (x86_64)

此操作將替換系統核心文件，存在以下風險：
1. 僅支持 x86_64 (amd64) 架構，其他架構將導致崩潰
2. 極少數情況下可能導致系統無法啟動 (Kernel Panic)
3. 請確保你擁有 VNC 控制台或救援模式權限`
		}

		warnBox := lipgloss.NewStyle().
			Padding(1, 1).
			Render(lipgloss.NewStyle().Foreground(style.Snow1).Render(strings.TrimSpace(warnText)))

		bottomDivider := lipgloss.NewStyle().
			Foreground(style.Polar4).
			Render(strings.Repeat("═", 50))

		bodyContent = lipgloss.JoinVertical(
			lipgloss.Center,
			"",
			warnHeader,
			warnBox,
			bottomDivider,
		)

	} else {
		// ===========================
		// 模式 B: 標準 BBR 菜單
		// ===========================

		desc := lipgloss.NewStyle().
			Foreground(style.Snow2).
			Render(" TCP 擁塞控制算法，提升網絡傳輸速度")

		// 1. 顯示 BBR 狀態
		labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
		valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)
		disabledStyle := lipgloss.NewStyle().Foreground(style.Muted)

		kernelVer := "未知"
		if info != nil {
			kernelVer = info.KernelVersion
		}

		var statusText string
		if info != nil && info.Enabled {
			statusText = fmt.Sprintf(
				" %s %s\n %s %s\n %s %s\n %s %s",
				labelStyle.Render("運行狀態:"),
				valueStyle.Render("✓ 已啟用"),
				labelStyle.Render("內核版本:"),
				lipgloss.NewStyle().Foreground(style.Snow3).Render(kernelVer),
				labelStyle.Render("BBR 類型:"),
				valueStyle.Render(info.Type),
				labelStyle.Render("擁塞算法:"),
				valueStyle.Render(info.Algorithm),
			)
		} else {
			statusText = fmt.Sprintf(" %s %s\n %s %s\n %s %s",
				labelStyle.Render("運行狀態:"),
				disabledStyle.Render("✗ 未啟用"),
				labelStyle.Render("內核版本:"),
				lipgloss.NewStyle().Foreground(style.Snow3).Render(kernelVer),
				labelStyle.Render("當前算法:"),
				lipgloss.NewStyle().Foreground(style.Snow3).Render("cubic (默認)"),
			)
		}

		// 2. 菜單選項
		items := []MenuItem{
			{"", "", "", lipgloss.Color("")},
			{constants.KeyBBR_Original, "啟用原版 BBR", "(Linux 內核 4.9+ 自帶)", style.StatusGreen},
			{constants.KeyBBR_BBR2, "啟用 BBR2", "(需內核支持，否則無效)", style.Aurora2},
			{constants.KeyBBR_XanMod, "安裝 XanMod 內核", "(BBRv3 高性能內核，需重啟)", style.Aurora3},
			{"", "", "", lipgloss.Color("")},
			{constants.KeyBBR_Disable, "禁用 BBR", "(恢復默認 cubic)", style.StatusRed},
		}
		menu := renderMenuWithAlignment(items, 0, "", false)

		// 3. 底部提示
		instruction := lipgloss.NewStyle().
			Foreground(style.Snow3).
			Render(" 💡 推薦：原版 BBR 穩定可靠，XanMod BBRv3 性能更強")

		warning := lipgloss.NewStyle().
			Foreground(style.StatusYellow).
			Render(`
 ⚠️  更換內核風險提示:
    可能導致系統無法啟動，請確保你已備份數據，
    並擁有 VNC 控制台或救援模式`)

		bodyContent = lipgloss.JoinVertical(
			lipgloss.Left,
			desc,
			divider,
			statusText,
			menu,
			"",
			instruction,
			warning,
		)
	}

	statusBlock := RenderStatusMessage(statusMsg)
	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		bodyContent,
		statusBlock,
		footer,
	)
}

package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderCoreMenu 渲染核心管理菜單
func RenderCoreMenu(
	coreVersion string,
	latestVersion string,
	hasUpdate bool,
	isInstalled bool,
	scriptVersion string,
	ti textinput.Model, // [CHANGE]
	statusMsg string,
) string {
	header := renderSubpageHeader("sing-box 核心管理")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 管理 sing-box 核心版本和更新")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 顯示版本信息
	labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)
	updateStyle := lipgloss.NewStyle().Foreground(style.StatusGreen).Bold(true)
	warnStyle := lipgloss.NewStyle().Foreground(style.StatusYellow)

	var versionText string
	if isInstalled {
		// [修改] 構建核心版本顯示字符串
		coreVerDisplay := valueStyle.Render(coreVersion)

		// 如果有更新，在後面追加顯示 "-> 發現新版: x.x.x"
		if latestVersion != "" && hasUpdate {
			arrow := lipgloss.NewStyle().Foreground(style.StatusYellow).Render("→")
			newVer := updateStyle.Render(fmt.Sprintf("發現新版: %s", latestVersion))
			coreVerDisplay = fmt.Sprintf("%s  %s %s", coreVerDisplay, arrow, newVer)
		} else if latestVersion != "" {
			// 如果已是最新，可以選擇不顯示或顯示 (已是最新)
			// coreVerDisplay += lipgloss.NewStyle().Foreground(style.Muted).Render(" (已是最新)")
		}

		displayScriptVer := scriptVersion
		if displayScriptVer != "" && !strings.HasPrefix(displayScriptVer, "v") {
			displayScriptVer = "v" + displayScriptVer
		}

		versionText = fmt.Sprintf(
			"%s %s\n%s %s",
			labelStyle.Render(" 當前版本："),
			coreVerDisplay,
			labelStyle.Render(" 腳本版本："),
			lipgloss.NewStyle().Foreground(style.Snow3).Render(displayScriptVer),
		)
	} else {
		versionText = warnStyle.Render(" ⚠️  sing-box 核心未安裝")
	}

	// 菜單項根據安裝狀態動態變化
	var items []MenuItem
	if isInstalled {
		items = []MenuItem{
			{"", "", "", lipgloss.Color("")},
			{constants.KeyCore_CheckUpdate, "檢查更新", "(檢測 sing-box 最新版本)", style.Aurora2},
			{constants.KeyCore_Update, "更新核心", "(升級到最新版本)", style.StatusGreen},
			{constants.KeyCore_Reinstall, "重新安裝", "(重新安裝當前版本)", style.Snow1},
			{constants.KeyCore_SelectVersion, "安裝指定版本", "(手動指定版本號安裝)", style.Snow1},
			{constants.KeyCore_SelectSource, "切換更新源", "(GitHub / 鏡像源)", style.Snow1},
			{"", "", "", lipgloss.Color("")},
			{constants.KeyCore_Uninstall, "卸載核心", "(移除 sing-box)", style.StatusRed},
			{"", "", "", lipgloss.Color("")},
			{constants.KeyScript_Update, "檢查腳本更新", "(檢查 Prism 更新)", style.Aurora2},
		}
	} else {
		items = []MenuItem{
			{"", "", "", lipgloss.Color("")},
			{constants.KeyCore_InstallLatest, "安裝最新版", "自動安裝最新穩定版", style.StatusGreen},
			{constants.KeyCore_SelectVersion, "安裝指定版本", "手動指定版本號安裝", style.Snow1},
			{constants.KeyCore_InstallDev, "安裝開發版", "安裝 beta/dev 版本", style.Snow1},
		}
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	var instruction string
	if hasUpdate && isInstalled {
		instruction = lipgloss.NewStyle().
			Foreground(style.StatusGreen).
			Render(" 🎉 有新版本可用，建議更新以獲得更好的性能和穩定性")
	} else if isInstalled {
		instruction = lipgloss.NewStyle().
			Foreground(style.Snow3).
			Render(" 💡 定期檢查更新以獲取最新功能和安全修復")
	} else {
		instruction = lipgloss.NewStyle().
			Foreground(style.StatusYellow).
			Render(" ⚠️  需要先安裝 sing-box 核心才能使用代理功能")
	}

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		versionText,
		menu,
		"",
		instruction,
		statusBlock,
		footer,
	)
}

package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderScriptUpdate 渲染腳本更新確認界面
func RenderScriptUpdate(
	currentVer string,
	latestVer string,
	changelog string,
	isChecking bool,
	ti textinput.Model,
	statusMsg string,
) string {
	header := renderSubpageHeader("管理腳本更新")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 查看更新內容並確認升級")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 內容區域
	var content string

	if isChecking {
		// 1. 檢查中狀態
		content = lipgloss.NewStyle().
			Foreground(style.Aurora3).
			Padding(2, 0).
			Render("正在連接 GitHub 獲取最新版本信息...")
	} else {
		// 2. 顯示結果狀態

		// 版本對比
		labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
		valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)
		newVerStyle := lipgloss.NewStyle().Foreground(style.StatusGreen)

		verStr := latestVer
		if latestVer == "" {
			verStr = "獲取失敗或無更新"
			newVerStyle = lipgloss.NewStyle().Foreground(style.StatusRed)
		}

		verBlock := fmt.Sprintf(
			"%s %s\n%s %s",
			labelStyle.Render(" 當前版本："),
			valueStyle.Render(currentVer),
			labelStyle.Render(" 最新版本："),
			newVerStyle.Render(verStr),
		)

		// 更新日誌 (帶邊框)
		logTitle := lipgloss.NewStyle().Foreground(style.Aurora1).Render("\n📄 更新內容 / Changelog :")

		logBoxStyle := lipgloss.NewStyle().
			Foreground(style.Snow1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(style.Polar3).
			Padding(0, 1).
			Width(65)

		if changelog == "" {
			changelog = "暫無詳細說明"
		}
		logBlock := logBoxStyle.Render(changelog)

		// 菜單
		menuItems := []MenuItem{
			{constants.KeyScriptUpdate_Confirm, "立即更新", "(下載並應用新版本)", style.StatusGreen},
			{constants.KeyScriptUpdate_Cancel, "取消返回", "(暫不更新)", style.Snow1},
		}

		// 如果沒有獲取到版本，只顯示返回
		if latestVer == "" {
			menuItems = []MenuItem{
				{constants.KeyScriptUpdate_Cancel, "返回菜單", "", style.Snow1},
			}
		}

		menu := renderMenuWithAlignment(menuItems, 0, "", false)

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			verBlock,
			logTitle,
			logBlock,
			"\n", // 空行
			menu,
		)
	}

	statusBlock := RenderStatusMessage(statusMsg)
	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		content,
		"",
		statusBlock,
		footer,
	)
}

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

// ========================================
// 渲染函數
// ========================================

// RenderUninstall 渲染卸載界面
func RenderUninstall(info *types.UninstallInfo, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("卸載 Prism")

	// 如果 info 為 nil，說明還在掃描中
	if info == nil {
		loading := lipgloss.NewStyle().
			Foreground(style.Aurora3).
			Padding(2, 0).
			Render(" ⏳ 正在掃描系統文件與佔用空間，請稍候...")

		return lipgloss.JoinVertical(lipgloss.Left, header, loading)
	}

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 移除程序及其組件，可選擇保留數據")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 信息概覽
	labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)

	statusText := fmt.Sprintf(
		" %s %s\n %s %s\n %s %s",
		labelStyle.Render("預計釋放空間："),
		valueStyle.Render(info.TotalSize),
		labelStyle.Render("日誌文件大小："),
		valueStyle.Render(info.LogSize),
		labelStyle.Render("備份文件數量："),
		valueStyle.Render(fmt.Sprintf("%d 個", info.BackupsCount)),
	)

	var content string

	if info.ConfirmStep == 0 {
		// ========================================================
		// 步驟 1: 選擇保留項
		// ========================================================
		on := lipgloss.NewStyle().Foreground(style.StatusGreen).Render("[保留]")
		off := lipgloss.NewStyle().Foreground(style.StatusRed).Render("[刪除]")

		state := func(keep bool) string {
			if keep {
				return on
			}
			return off
		}

		items := []MenuItem{
			{constants.KeyUninstall_KeepConfig, "配置文件", fmt.Sprintf("%s %s", state(info.KeepConfig), info.ConfigPath), style.Snow1},
			{constants.KeyUninstall_KeepCert, "證書文件", fmt.Sprintf("%s %s", state(info.KeepCerts), info.CertDir), style.Snow1},
			{constants.KeyUninstall_KeepBackup, "備份文件", fmt.Sprintf("%s %s", state(info.KeepBackups), info.BackupDir), style.Snow1},
			{constants.KeyUninstall_KeepLog, "日誌文件", fmt.Sprintf("%s %s", state(info.KeepLogs), info.LogDir), style.Snow1},
			{"", "", "", lipgloss.Color("")},
			{constants.KeyUninstall_ConfirmStep, "下一步", "進入最終確認", style.Aurora2},
		}

		menu := renderMenuWithAlignment(items, 0, "", false)
		instruction := lipgloss.NewStyle().Foreground(style.Snow3).Render(" 💡 輸入選項編號切換保留狀態")
		content = lipgloss.JoinVertical(lipgloss.Left, menu, "", instruction)

	} else {
		// ========================================================
		// 步驟 2: 最終確認 (輸入 UNINSTALL)
		// ========================================================
		confirmTitle := lipgloss.NewStyle().
			Foreground(style.StatusRed).
			Bold(true).
			Render("\n ⚠️  危險操作確認")

		warnStyle := lipgloss.NewStyle().Foreground(style.StatusRed)
		snowStyle := lipgloss.NewStyle().Foreground(style.Snow2)

		// 構建對齊的操作列表
		var opLines []string
		opLines = append(opLines, "• 停止並禁用 sing-box 服務")
		opLines = append(opLines, "• 移除 sing-box 核心程序")

		if !info.KeepConfig {
			opLines = append(opLines, "• 刪除所有配置文件")
		}
		if !info.KeepCerts {
			opLines = append(opLines, "• 刪除所有證書")
		}
		if !info.KeepBackups {
			opLines = append(opLines, "• 刪除所有備份")
		}
		if !info.KeepLogs {
			opLines = append(opLines, "• 刪除所有日誌")
		}

		headerText := snowStyle.Render(" 即將執行的操作：")

		// 統一渲染操作列表
		opsContent := warnStyle.Render(strings.Join(opLines, "\n"))
		opsContent = lipgloss.NewStyle().PaddingLeft(3).Render(opsContent)

		deleteText := lipgloss.JoinVertical(lipgloss.Left, headerText, opsContent)

		yellowStyle := lipgloss.NewStyle().Foreground(style.StatusYellow)
		keywordStyle := lipgloss.NewStyle().Foreground(style.StatusRed)

		warningText := "\n" +
			yellowStyle.Render(" 此操作不可逆！請在下方輸入 ") +
			keywordStyle.Render("UNINSTALL") +
			yellowStyle.Render(" 確認卸載")

		divider2 := lipgloss.NewStyle().
			Foreground(style.Polar4).
			Render(strings.Repeat("═", 50))

		content = lipgloss.JoinVertical(lipgloss.Left, confirmTitle, "", deleteText, warningText, "", divider2)
	}

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		statusText,
		divider,
		content,
		statusBlock,
		footer,
	)
}

// RenderUninstallProgress 渲染卸載進度
func RenderUninstallProgress(steps []types.UninstallStep, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader(" 卸載進行中")

	var stepLines []string
	for _, step := range steps {
		var icon string
		var color lipgloss.Color

		switch step.Status {
		case "success":
			icon = "✓"
			color = style.StatusGreen
		case "failed":
			icon = "✗"
			color = style.StatusRed
		case "running":
			icon = "◉"
			color = style.StatusYellow
		default: // pending
			icon = "○"
			color = style.Muted
		}

		statusStyle := lipgloss.NewStyle().Foreground(color)
		line := fmt.Sprintf("%s %s", statusStyle.Render(icon), step.Name)

		if step.Message != "" {
			line += " - " + lipgloss.NewStyle().Foreground(style.Snow3).Render(step.Message)
		}
		stepLines = append(stepLines, line)
	}

	content := strings.Join(stepLines, "\n")

	statusBlock := RenderStatusMessage(statusMsg)

	return lipgloss.JoinVertical(lipgloss.Left, header, "", content, statusBlock)
}

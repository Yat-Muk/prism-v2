package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// RenderInstallProgress 渲染安裝進度
func RenderInstallProgress(logs []string, spinnerModel spinner.Model) string {
	// 1. 渲染頭部 (寬度約 50)
	header := renderSubpageHeader("系統安裝中")

	// 2. 定義樣式
	// 基礎文字樣式
	textStyle := lipgloss.NewStyle().Foreground(style.Snow2)
	// 高亮樣式 (正在進行的步驟)
	activeStyle := lipgloss.NewStyle().Foreground(style.Snow1).Bold(true)
	// 成功標記樣式 (綠色 ✓)
	checkStyle := lipgloss.NewStyle().Foreground(style.StatusGreen).Bold(true)
	// 失敗標記樣式 (紅色 ✗)
	errorStyle := lipgloss.NewStyle().Foreground(style.StatusRed).Bold(true)
	// 按鍵樣式 (綠色 [Enter])
	keyStyle := lipgloss.NewStyle().Foreground(style.StatusGreen).Bold(true)
	// 邊框/容器樣式 (與主菜單寬度一致，左對齊)
	containerStyle := lipgloss.NewStyle().
		Width(50).            // 與分隔線寬度一致
		Align(lipgloss.Left). // 強制左對齊
		PaddingTop(1)

	// 3. 構建日誌視圖
	var logLines []string

	// 只顯示最後 10 行，避免過長
	displayLogs := logs
	if len(logs) > 10 {
		displayLogs = logs[len(logs)-10:]
	}

	for i, log := range displayLogs {
		isLast := (i == len(displayLogs)-1)

		// === 最後一行 (活躍狀態) ===
		if isLast {
			var rowContent string

			// 特殊處理包含 [Enter] 的行，防止樣式重置導致後半段顏色丟失
			if strings.Contains(log, "[Enter]") {
				parts := strings.Split(log, "[Enter]")
				prefix := activeStyle.Render(parts[0])
				key := keyStyle.Render("[Enter]")
				suffix := ""
				if len(parts) > 1 {
					suffix = activeStyle.Render(parts[1])
				}
				rowContent = prefix + key + suffix
			} else {
				rowContent = activeStyle.Render(log)
			}

			row := fmt.Sprintf(" %s%s", spinnerModel.View(), rowContent)
			logLines = append(logLines, row)

		} else {
			// === 已完成的步驟 (歷史記錄) ===

			var prefix string
			if strings.Contains(log, "失敗") ||
				strings.Contains(log, "Error") ||
				strings.Contains(log, "Err") {
				// 失敗：顯示紅色 ✗
				prefix = errorStyle.Render(" ✗")
			} else {
				// 成功：顯示綠色 ✓
				prefix = checkStyle.Render(" ✓")
			}

			// 處理歷史記錄中的 [Enter] 樣式 (雖然少見，但保持一致性)
			displayText := log
			if strings.Contains(log, "[Enter]") {
				displayText = strings.ReplaceAll(log, "[Enter]", "Enter")
			}

			row := fmt.Sprintf("%s %s", prefix, textStyle.Render(displayText))
			logLines = append(logLines, row)
		}
	}

	// 補齊高度
	minHeight := 8
	if len(logLines) < minHeight {
		logLines = append(logLines, strings.Repeat("\n", minHeight-len(logLines)))
	}

	// 4. 底部提示
	footer := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Italic(true).
		Width(50).
		Align(lipgloss.Center).
		Render("請勿關閉程序，正在自動配置環境...")

	content := containerStyle.Render(strings.Join(logLines, "\n"))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		"\n",
		footer,
	)
}

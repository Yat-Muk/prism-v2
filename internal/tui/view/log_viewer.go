package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// LogViewerMode 日誌查看模式
type LogViewerMode string

const (
	LogViewerModeRealtime LogViewerMode = "realtime"
	LogViewerModeFull     LogViewerMode = "full"
	LogViewerModeError    LogViewerMode = "error"
)

// RenderLogViewer 渲染日誌查看器 (集成 Viewport)
func RenderLogViewer(mode LogViewerMode, logs []string, vp viewport.Model, following bool) string {
	var title string
	switch mode {
	case LogViewerModeRealtime:
		title = "實時日誌"
	case LogViewerModeFull:
		title = "完整日誌"
	case LogViewerModeError:
		title = "錯誤日誌"
	default:
		title = "日誌查看器"
	}

	header := renderSubpageHeader(title)

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("═", 50))

	// --- 內容處理 ---
	// 如果 logs 不為空，說明有新內容傳入，這部分通常在 Update 中處理
	// 這裡我們主要渲染 vp.View()

	// 使用 Viewport 渲染內容區
	// 注意：Viewport 的內容設置應該在 Update 階段完成，這裡只負責佈局
	content := vp.View()

	// 如果內容為空，顯示提示
	if content == "" {
		content = lipgloss.NewStyle().
			Foreground(style.Muted).
			Padding(1, 1).
			Render("暫無日誌數據或加載中...")
	}

	// 狀態欄
	statusStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	var statusText string

	percent := int(vp.ScrollPercent() * 100)

	if following {
		statusText = statusStyle.Render(fmt.Sprintf(" 📡 實時跟蹤中... | 進度: %d%% (按 Esc 返回)", percent))
	} else {
		statusText = statusStyle.Render(fmt.Sprintf(" 按 Esc 返回 | ↑/↓ 滾動 | PgUp/PgDn 翻頁 | 進度: %d%%", percent))
	}

	footer := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 提示：日誌已自動截取最新部分以優化性能")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content, // 這裡放置滾動視窗
		divider,
		"",
		statusText,
		"",
		footer,
	)
}

// Helper: 構建帶顏色的日誌字符串
func BuildColoredLogContent(logs []string) string {
	if len(logs) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, log := range logs {
		if strings.Contains(log, "ERROR") || strings.Contains(log, "FATAL") {
			sb.WriteString(lipgloss.NewStyle().Foreground(style.StatusRed).Render(log))
		} else if strings.Contains(log, "WARN") {
			sb.WriteString(lipgloss.NewStyle().Foreground(style.StatusYellow).Render(log))
		} else if strings.Contains(log, "DEBUG") {
			sb.WriteString(lipgloss.NewStyle().Foreground(style.Muted).Render(log))
		} else {
			sb.WriteString(lipgloss.NewStyle().Foreground(style.Snow1).Render(log))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderCertDelete 渲染证书删除界面
func RenderCertDelete(domains []string, confirmMode bool, toDelete string, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("刪除證書")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 選擇要刪除的證書（謹慎操作）")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 确认模式
	if confirmMode {
		warning := lipgloss.NewStyle().
			Foreground(style.StatusRed).
			Render(fmt.Sprintf(`
 ⚠️  警告：您即將刪除證書 %s

 此操作將刪除：
  • 證書文件 (%s.crt)
  • 私鑰文件 (%s.key)
  • 元數據文件 (%s.json)

 此操作不可恢復！
`, toDelete, toDelete, toDelete, toDelete))

		instruction := lipgloss.NewStyle().
			Foreground(style.Snow2).
			Render(fmt.Sprintf(" 輸入 %s 確認刪除，或按 Esc 取消",
				lipgloss.NewStyle().Foreground(style.StatusRed).Render("YES")))

		statusBlock := RenderStatusMessage(statusMsg)

		footer := RenderInputFooter(ti)

		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			desc,
			divider,
			warning,
			instruction,
			statusBlock,
			footer,
		)
	}

	// 选择模式
	items := []MenuItem{}

	for i, domain := range domains {
		items = append(items, MenuItem{
			Num:       fmt.Sprintf("%d", i+1),
			Text:      domain,
			Desc:      "(將永久刪除)",
			TextColor: style.StatusRed,
		})
	}

	if len(domains) == 0 {
		items = append(items, MenuItem{
			Num:       "",
			Text:      "暫無可刪除的證書",
			Desc:      "",
			TextColor: style.Muted,
		})
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := style.WarningText(" ⚠️  請選擇要刪除的證書（此操作不可恢復）")

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		menu,
		"",
		instruction,
		statusBlock,
		footer,
	)
}

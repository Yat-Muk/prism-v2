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

func RenderBackupRestoreMenu(
	cursor int,
	backups []types.BackupItem,
	ti textinput.Model,
	statusMsg string,
	selected string,
	confirmMode bool,
	pendingOp string,
) string {
	header := renderSubpageHeader("配置備份恢復")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 功能：備份/恢復 Prism 配置，支持快速回滾最近備份")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 備份記錄區
	record := renderBackupRecordBlock(backups)

	items := []MenuItem{
		{"", "", "", lipgloss.Color("")},
		{constants.KeyBackup_Create, "備份當前配置", " (立即備份 YAML 到備份目錄)", style.Snow1},
		{constants.KeyBackup_Restore, "一鍵恢復備份", " (進入恢復模式選擇文件)", style.StatusYellow},
		{constants.KeyBackup_Delete, "刪除歷史備份", " (清理舊的備份文件)", style.StatusRed},
	}

	// 處理確認模式 (如果不加這段，確認界面無法顯示)
	var menuContent string
	if confirmMode {
		var actionText string
		var colorStyle lipgloss.Color

		if pendingOp == "delete" {
			actionText = "刪除"
			colorStyle = style.StatusRed
		} else {
			actionText = "恢復"
			colorStyle = style.StatusYellow
		}

		warnBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorStyle).
			Padding(1, 2).
			Render(fmt.Sprintf("⚠️  確定要 %s 選中的備份嗎？\n   目標: %s\n\n   [YES] 確認   [Esc] 取消", actionText, selected))

		menuContent = "\n" + warnBox + "\n"
	} else {
		menuContent = renderMenuWithAlignment(items, -1, "", false)
	}

	statusBlock := RenderStatusMessage(statusMsg)
	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		record,
		menuContent,
		statusBlock,
		footer,
	)
}

func renderBackupRecordBlock(backups []types.BackupItem) string {
	var lines []string
	titleStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	title := fmt.Sprintf(" 最近備份記錄 (共 %d 個)", len(backups))
	lines = append(lines, titleStyle.Render(title))
	lines = append(lines, lipgloss.NewStyle().Foreground(style.Polar4).Render(" "+strings.Repeat("┄", 48)))

	if len(backups) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(style.Muted).Render(" (暫無備份)"))
	} else {
		maxCount := 3
		if len(backups) < maxCount {
			maxCount = len(backups)
		}
		for i := 0; i < maxCount; i++ {
			b := backups[i]
			displayName := b.Name
			displayName = strings.TrimPrefix(displayName, "config-")
			displayName = strings.TrimSuffix(displayName, ".yaml")
			sizeStr := fmt.Sprintf("%.1fKB", float64(b.Size)/1024)
			line := fmt.Sprintf(" %d. %s (%s)", i+1, displayName, sizeStr)
			lines = append(lines, lipgloss.NewStyle().Foreground(style.Aurora2).Render(line))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

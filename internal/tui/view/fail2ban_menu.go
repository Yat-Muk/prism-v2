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

// RenderFail2BanMenu 渲染 Fail2Ban 防護菜單
func RenderFail2BanMenu(info *types.Fail2BanInfo, list []string, inputMode bool, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("Fail2Ban 防護")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	var bodyContent string

	if inputMode {
		listHeader := lipgloss.NewStyle().
			Foreground(style.Aurora2).
			Bold(true).
			Render("--- 當前封禁名單 (請輸入 IP 解封) ---")

		var listStr string
		if len(list) == 0 {
			listStr = lipgloss.NewStyle().Foreground(style.Muted).Render("   (正在獲取列表...)")
		} else {
			var cleanList []string
			for _, line := range list {
				if strings.TrimSpace(line) != "" {
					cleanList = append(cleanList, "   "+line)
				}
			}
			if len(cleanList) > 15 {
				cleanList = cleanList[len(cleanList)-15:]
			}
			listStr = strings.Join(cleanList, "\n")
		}

		bottomDivider := lipgloss.NewStyle().
			Foreground(style.Polar4).
			Render(strings.Repeat("═", 50))

		bodyContent = fmt.Sprintf("\n%s\n\n%s\n\n%s", listHeader, listStr, bottomDivider)

	} else {
		desc := lipgloss.NewStyle().
			Foreground(style.Snow2).
			Render(" SSH 暴力破解防護系統")

		labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
		valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)
		disabledStyle := lipgloss.NewStyle().Foreground(style.Muted)
		greenStyle := lipgloss.NewStyle().Foreground(style.StatusGreen)
		redStyle := lipgloss.NewStyle().Foreground(style.StatusRed)

		var statusText string
		if info != nil && info.Installed {
			var runningStatus string
			if info.Running {
				runningStatus = greenStyle.Render("✓ 運行中")
			} else {
				runningStatus = redStyle.Render("✗ 未運行")
			}

			statusText = fmt.Sprintf(
				" %s %s\n %s %s\n %s %d\n %s %d\n %s %d 次\n %s %s",
				labelStyle.Render("安裝狀態:"),
				valueStyle.Render("✓ 已安裝"),
				labelStyle.Render("運行狀態:"),
				runningStatus,
				labelStyle.Render("已封禁 IP:"),
				info.BannedIPs,
				labelStyle.Render("SSH 攻擊次數:"),
				info.SSHAttempts,
				labelStyle.Render("最大重試次數:"),
				info.MaxRetry,
				labelStyle.Render("封禁時長:"),
				valueStyle.Render(info.BanTime),
			)
		} else {
			statusText = fmt.Sprintf(" %s %s",
				labelStyle.Render("狀態:"),
				disabledStyle.Render("✗ 未安裝"))
		}

		items := []MenuItem{
			{"", "", "", lipgloss.Color("")},
			{constants.KeyFail2Ban_Install, "安裝 Fail2Ban", "(安裝並啟動防護)", style.StatusGreen},
			{constants.KeyFail2Ban_Toggle, "啟動/停止", "(控制 Fail2Ban 服務)", style.Snow1},
			{constants.KeyFail2Ban_List, "查看封禁 IP", "(顯示被封禁的 IP 列表)", style.Aurora2},
			{constants.KeyFail2Ban_Unban, "解封 IP", "(解除指定 IP 的封禁)", style.StatusYellow},
			{constants.KeyFail2Ban_Config, "配置規則", "(修改封禁策略)", style.Snow1},
			{"", "", "", lipgloss.Color("")},
			{constants.KeyFail2Ban_Uninstall, "卸載", "(移除 Fail2Ban)", style.StatusRed},
		}
		menu := renderMenuWithAlignment(items, 0, "", false)

		instruction := lipgloss.NewStyle().
			Foreground(style.Snow3).
			Render(" 💡 推薦：SSH 密鑰登錄 + Fail2Ban 雙重保護")

		bodyContent = lipgloss.JoinVertical(
			lipgloss.Left,
			desc,
			divider,
			statusText,
			menu,
			"",
			instruction,
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

// RenderFail2BanList 專門用於渲染封禁列表
func RenderFail2BanList(lines []string, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("Fail2Ban 封禁名單")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("═", 50))

	content := strings.Join(lines, "\n")

	if len(lines) < 10 {
		content += strings.Repeat("\n", 10-len(lines))
	}

	contentStyle := lipgloss.NewStyle().
		Foreground(style.Snow1).
		PaddingLeft(2)

	statusBlock := RenderStatusMessage(statusMsg)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" Esc 返回")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		contentStyle.Render(content),
		statusBlock,
		divider,
		"",
		instruction,
	)
}

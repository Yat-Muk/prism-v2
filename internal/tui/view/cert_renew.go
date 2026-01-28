package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderCertRenew 渲染证书续期界面
func RenderCertRenew(domains []string, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("證書續期")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 選擇要續期的證書")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 构建菜单项
	items := []MenuItem{
		{constants.KeyRenew_All, "續期所有證書", "(檢查所有證書並續期即將過期的)", style.StatusYellow},
	}

	// 添加单个域名选项
	// 注意：Handler 邏輯假定 KeyRenew_All 為 "1"，因此動態項從 2 開始
	startIdx := 2
	for i, domain := range domains {
		items = append(items, MenuItem{
			Num:       fmt.Sprintf("%d", i+startIdx),
			Text:      domain,
			Desc:      "(單獨續期此證書)",
			TextColor: style.Aurora1,
		})
	}

	if len(domains) == 0 {
		items = append(items, MenuItem{
			Num:       "",
			Text:      "暫無 ACME 證書",
			Desc:      "",
			TextColor: style.Muted,
		})
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := style.InfoText(" 請選擇要續期的證書")

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

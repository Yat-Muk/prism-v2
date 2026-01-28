package view

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/domain/protocol"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderPortEdit 渲染端口編輯頁
func RenderPortEdit(enabledProtocols []int, currentPorts map[int]int, hy2Hopping string, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("修改監聽端口")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 修改 Sing-box 各協議的監聽端口")

	infoSep := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	infoBlock := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		infoSep,
	)

	sortedProtocols := make([]int, len(enabledProtocols))
	copy(sortedProtocols, enabledProtocols)
	sort.Ints(sortedProtocols)

	// 定義樣式
	portStyle := lipgloss.NewStyle().Foreground(style.Aurora4) // 橙色
	hintStyle := lipgloss.NewStyle().Foreground(style.Muted)   // 灰色

	// 生成 MenuItem
	var items []MenuItem

	// 遍歷已啟用的協議 ID
	for i, protoID := range sortedProtocols {
		pID := protocol.ID(protoID)
		if !pID.IsValid() {
			continue
		}

		// 處理主端口顯示
		portStr := "未設置"
		if v, ok := currentPorts[protoID]; ok && v > 0 {
			portStr = fmt.Sprintf("%d", v)
		}

		// 構造描述字符串：[橙色端口]
		descText := portStyle.Render(portStr)

		// 動態處理提示信息
		if pID == protocol.IDHysteria2 {
			hintText := ""
			if hy2Hopping != "" {
				// 如果有跳躍範圍，顯示具體範圍
				hintText = fmt.Sprintf("(%s)", hy2Hopping)
			} else {
				// 否則顯示默認提示
				hintText = "(支持跳躍端口)"
			}
			descText += " " + hintStyle.Render(hintText)
		}

		items = append(items, MenuItem{
			Num:       fmt.Sprintf("%d", i+1),
			Text:      pID.String(),
			Desc:      descText,
			TextColor: style.Snow1,
		})
	}

	// 添加重置選項
	items = append(items,
		MenuItem{}, // 空行分隔
		MenuItem{
			Num:       "r",
			Text:      "隨機重置所有端口",
			Desc:      hintStyle.Render(""),
			TextColor: style.StatusRed,
		},
	)

	menu := renderMenuWithAlignment(items, 0, "", true)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(" 💡 輸入協議編號修改單個端口，或輸入 r 重置所有")

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		infoBlock,
		menu,
		"",
		instruction,
		statusBlock,
		footer,
	)
}

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

// RenderProtocolLinks 渲染協議列表 (支持 查看鏈接 和 選擇二維碼 兩種模式)
func RenderProtocolLinks(links []types.ProtocolLink, mode string, ti textinput.Model, statusMsg string) string {
	var (
		headerTitle string
		descText    string
		showCopyAll bool
	)

	if mode == "qrcode" {
		headerTitle = "生成二維碼"
		descText = " 請輸入序號選擇協議（單選）"
		showCopyAll = false
	} else {
		headerTitle = "協議鏈接"
		descText = " 客戶端連接鏈接（支持多選/批量複製）"
		showCopyAll = true
	}

	header := renderSubpageHeader(headerTitle)
	desc := lipgloss.NewStyle().Foreground(style.Gray).Render(descText)
	divider := lipgloss.NewStyle().Foreground(style.Gray).Render(strings.Repeat("─", 50))

	infoBlock := lipgloss.JoinVertical(lipgloss.Left, header, desc, divider)

	var listContent string
	if len(links) == 0 {
		listContent = lipgloss.NewStyle().
			Foreground(style.Gray).
			PaddingLeft(2).
			Render("暫無可用協議鏈接\n請先在配置管理中啓用協議")
	} else {
		var items []MenuItem

		portStyle := lipgloss.NewStyle().Foreground(style.Orange)
		metaStyle := lipgloss.NewStyle().Foreground(style.Gray)

		for i, link := range links {
			descText := fmt.Sprintf("%s %s%s",
				metaStyle.Render("(端口:"),
				portStyle.Render(fmt.Sprintf("%d", link.Port)),
				metaStyle.Render(")"),
			)

			items = append(items, MenuItem{
				Num:       fmt.Sprintf("%d", i+1),
				Text:      link.Name,
				Desc:      descText,
				TextColor: style.SkyBlue,
			})
		}

		if showCopyAll {
			items = append(items,
				MenuItem{},
				MenuItem{
					Num:       constants.KeyNode_Copy,
					Text:      "複製所有鏈接",
					Desc:      metaStyle.Render("(批量導出到剪貼板)"),
					TextColor: style.FutureGreen,
				},
			)
		}

		listContent = renderMenuWithAlignment(items, 0, "", true)
	}

	// 底部提示
	var instruction string
	hintStyle := lipgloss.NewStyle().Foreground(style.Gray)

	if mode == "qrcode" {
		instruction = hintStyle.Render(" 💡 輸入數字序號查看二維碼")
	} else {
		instruction = hintStyle.Render(" 💡 輸入序號複製鏈接（可多選，如 2,5,7）")
	}

	statusBlock := RenderStatusMessage(statusMsg)
	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		infoBlock,
		listContent,
		"",
		instruction,
		statusBlock,
		footer,
	)
}

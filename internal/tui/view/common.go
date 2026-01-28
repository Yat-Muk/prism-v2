package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// MenuItem 菜單項結構
type MenuItem struct {
	Num       string         // 序號 (如 "1", "r")
	Text      string         // 選項名稱 (如 "Hysteria 2")
	Desc      string         // 描述/提示 (如 "(支持跳躍)") -> 自動渲染為灰色
	TextColor lipgloss.Color // Text 的顏色
}

// renderMenuWithAlignment 渲染自動對齊的菜單列表
func renderMenuWithAlignment(items []MenuItem, cursor int, filter string, isTableMode bool) string {
	maxNumWidth := 0
	maxTextWidth := 0

	for _, item := range items {
		// 跳過分隔線
		if item.Num == "" && item.Text == "" {
			continue
		}

		// 1. 計算序號寬度
		if len(item.Num) > maxNumWidth {
			maxNumWidth = len(item.Num)
		}

		// 2. 計算文本寬度 (根據模式決定誰參與計算)
		w := runewidth.StringWidth(item.Text)

		// 根據模式決定是否更新 maxTextWidth
		updateMax := false
		if isTableMode {
			// 表格模式 (端口頁)：所有項目參與，確保最長標題撐開佈局
			updateMax = true
		} else {
			// 菜單模式 (主菜單)：只有帶括號的項目參與，防止無括號長標題破壞對齊
			if strings.Contains(item.Desc, "(") || strings.Contains(item.Desc, "（") {
				updateMax = true
			}
		}

		if updateMax && w > maxTextWidth {
			maxTextWidth = w
		}
	}

	// 設定統一的目標寬度 = 最長文本 + 2個空格間距
	targetWidth := 0
	if maxTextWidth > 0 {
		targetWidth = maxTextWidth + 2
	}

	var rows []string
	for _, item := range items {
		// 處理分隔線
		if item.Num == "" && item.Text == "" {
			separator := lipgloss.NewStyle().
				Foreground(style.Snow2).
				Render(" " + strings.Repeat("┄", 48))
			rows = append(rows, separator)
			continue
		}

		// 定義樣式
		numStyle := lipgloss.NewStyle().Foreground(style.Aurora3)
		textStyle := lipgloss.NewStyle().Foreground(item.TextColor)
		dotStyle := lipgloss.NewStyle().Foreground(style.Snow3)

		// 渲染序號
		prefix := " "
		numStr := fmt.Sprintf("%*s", maxNumWidth, item.Num)
		dotStr := dotStyle.Render(".")

		// 渲染文本與填充
		nameText := item.Text
		currentWidth := runewidth.StringWidth(nameText)
		padding := " " // 默認最小間距

		// 判斷當前行是否需要執行對齊
		shouldAlign := false

		if isTableMode {
			// 【表格模式】強制所有行對齊，確保右側端口號垂直排成一條線
			shouldAlign = true
		} else {
			// 【菜單模式】只對齊帶括號的行
			if strings.Contains(item.Desc, "(") || strings.Contains(item.Desc, "（") {
				shouldAlign = true
			}
		}

		if shouldAlign && targetWidth > 0 {
			gap := targetWidth - currentWidth
			if gap < 1 {
				gap = 1 // 至少保留 1 個空格，防止文字粘連
			}
			padding = strings.Repeat(" ", gap)
		}

		displayName := textStyle.Render(nameText) + padding

		// 處理描述顏色
		var descDisplay string
		if strings.Contains(item.Desc, "\x1b") {
			descDisplay = item.Desc
		} else {
			descDisplay = colorizeDescription(item.Desc)
		}

		// 組合最終行
		row := fmt.Sprintf("%s%s%s %s%s",
			prefix,
			numStyle.Render(numStr),
			dotStr,
			displayName,
			descDisplay,
		)
		rows = append(rows, row)
	}

	// 底部雙線
	bottomSeparator := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(strings.Repeat("═", 50))
	rows = append(rows, bottomSeparator)

	return strings.Join(rows, "\n")
}

// colorizeDescription 默認著色邏輯：括號變灰，中括號變黃
func colorizeDescription(desc string) string {
	if desc == "" {
		return ""
	}

	yellowStyle := lipgloss.NewStyle().Foreground(style.StatusYellow)
	greyStyle := lipgloss.NewStyle().Foreground(style.Snow3)

	var result strings.Builder
	runes := []rune(desc)
	n := len(runes)

	for i := 0; i < n; i++ {
		char := runes[i]
		if char == '[' {
			// [內容] -> 黃色
			start := i
			for i < n && runes[i] != ']' {
				i++
			}
			if i < n {
				result.WriteString(yellowStyle.Render(string(runes[start : i+1])))
			} else {
				result.WriteString(greyStyle.Render(string(runes[start:])))
			}
		} else if char == '(' {
			// (內容) -> 灰色
			start := i
			for i < n && runes[i] != ')' {
				i++
			}
			if i < n {
				result.WriteString(greyStyle.Render(string(runes[start : i+1])))
			} else {
				result.WriteString(greyStyle.Render(string(runes[start:])))
			}
		} else {
			// 普通文字 -> 灰色 (默認)
			start := i
			for i < n && runes[i] != '[' && runes[i] != '(' {
				i++
			}
			result.WriteString(greyStyle.Render(string(runes[start:i])))
			i--
		}
	}
	return result.String()
}

// RenderLogo 渲染 PRISM ASCII Logo
func RenderLogo() string {
	logoLines := []string{
		" ██████╗ ██████╗ ██╗███████╗███╗   ███╗",
		" ██╔══██╗██╔══██╗██║██╔════╝████╗ ████║",
		" ██████╔╝██████╔╝██║███████╗██╔████╔██║",
		" ██╔═══╝ ██╔══██╗██║╚════██║██║╚██╔╝██║",
		" ██║     ██║  ██║██║███████║██║ ╚═╝ ██║",
		" ╚═╝     ╚═╝  ╚═╝╚═╝╚══════╝╚═╝     ╚═╝",
	}

	gradientColors := []lipgloss.Color{
		lipgloss.Color("#B477ED"),
		lipgloss.Color("#DDAAFF"),
		lipgloss.Color("#DEDEF8"),
		lipgloss.Color("#90CCFB"),
		lipgloss.Color("#1AAEFC"),
		lipgloss.Color("#0381ED"),
	}

	var coloredLines []string
	for i, line := range logoLines {
		coloredLine := lipgloss.NewStyle().
			Foreground(gradientColors[i]).
			Width(50).
			AlignHorizontal(lipgloss.Center).
			Render(line)
		coloredLines = append(coloredLines, coloredLine)
	}

	return lipgloss.JoinVertical(lipgloss.Left, coloredLines...)
}

// renderSubpageHeader 渲染子頁面頭部
func renderSubpageHeader(subTitle string) string {
	logo := RenderLogo()

	mainSubtitle := lipgloss.NewStyle().
		Foreground(style.Aurora3).
		Width(50).
		AlignHorizontal(lipgloss.Center).
		Render(":: 現代化 sing-box 管理工具 ::")

	subTitleLine := lipgloss.NewStyle().
		Foreground(style.Aurora2).
		Render(fmt.Sprintf(" »»» %s «««", subTitle))

	// 雙線分隔
	separator := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(strings.Repeat("═", 50))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		logo,
		"",
		mainSubtitle,
		"",
		subTitleLine,
		separator,
	)
}

// RenderStatusMessage 通用狀態提示渲染器
// 自動根據關鍵字（⚠️, 失敗, 成功）決定顏色
// RenderStatusMessage 渲染底部狀態欄 (修復版)
func RenderStatusMessage(msg string) string {
	if msg == "" {
		return ""
	}

	// 1. 確定基礎顏色
	baseColor := style.Aurora3
	if strings.Contains(msg, "⚠️") ||
		strings.Contains(msg, "重置") ||
		strings.Contains(msg, "警告") {
		baseColor = style.StatusYellow
	} else if strings.Contains(msg, "失敗") ||
		strings.Contains(msg, "錯誤") ||
		strings.Contains(msg, "無效") ||
		strings.Contains(msg, "✗") {
		baseColor = style.StatusRed
	} else if strings.Contains(msg, "成功") ||
		strings.Contains(msg, "完成") ||
		strings.Contains(msg, "✓") {
		baseColor = style.StatusGreen
	}

	baseStyle := lipgloss.NewStyle().Foreground(baseColor)
	highlightStyle := lipgloss.NewStyle().Foreground(style.StatusRed)

	keywords := []string{"(Y/N)", "YES", "UNINSTALL"}

	// 2. 先按換行符切分，單獨處理每一行
	rawLines := strings.Split(msg, "\n")
	var renderedLines []string

	type segment struct {
		text   string
		isHigh bool
	}

	for _, line := range rawLines {
		// 對每一行單獨執行高亮邏輯
		segments := []segment{{text: line, isHigh: false}}

		for _, kw := range keywords {
			var newSegments []segment
			for _, seg := range segments {
				if seg.isHigh {
					newSegments = append(newSegments, seg)
					continue
				}
				parts := strings.Split(seg.text, kw)
				for i, part := range parts {
					if part != "" {
						newSegments = append(newSegments, segment{text: part, isHigh: false})
					}
					if i < len(parts)-1 {
						newSegments = append(newSegments, segment{text: kw, isHigh: true})
					}
				}
			}
			segments = newSegments
		}

		// 拼接當前行
		var sb strings.Builder
		for _, seg := range segments {
			if seg.isHigh {
				sb.WriteString(highlightStyle.Render(seg.text))
			} else {
				sb.WriteString(baseStyle.Render(seg.text))
			}
		}
		renderedLines = append(renderedLines, sb.String())
	}

	// 3. 使用 JoinVertical 組合所有行
	content := lipgloss.JoinVertical(lipgloss.Left, renderedLines...)

	// 4. 包裝外框
	return lipgloss.NewStyle().
		Padding(1, 1).
		Width(52).
		Align(lipgloss.Left).
		Render(content)
}

// 只渲染輸入行，不帶底部按鍵提示（供 Main View 使用）
func RenderTextInput(ti textinput.Model) string {
	prompt := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" ❯ 請輸入: ")

	return lipgloss.JoinHorizontal(lipgloss.Left, prompt, ti.View())
}

// RenderInputFooter 渲染輸入提示（子菜單使用）
func RenderInputFooter(ti textinput.Model) string {
	prompt := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" ❯ 請輸入: ")

	inputLine := lipgloss.JoinHorizontal(lipgloss.Left, prompt, ti.View())

	snow3 := lipgloss.NewStyle().Foreground(style.Snow3)
	polar4 := lipgloss.NewStyle().Foreground(style.Polar4)

	hints := lipgloss.JoinHorizontal(lipgloss.Left,
		snow3.Render("Esc "), polar4.Render("返回"),
		polar4.Render(" • "),
		snow3.Render("Enter "), polar4.Render("確認"),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		// "",
		inputLine,
		"\n",
		lipgloss.NewStyle().PaddingLeft(1).Render(hints),
	)
}

// RenderError 渲染錯誤頁面
func RenderError(errMsg string, ti textinput.Model) string {
	header := renderSubpageHeader("錯誤")
	errorStyle := lipgloss.NewStyle().Foreground(style.StatusRed).Bold(true)
	errorText := errorStyle.Render(fmt.Sprintf("✗ %s", errMsg))

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(lipgloss.Left, header, "", errorText, "", footer)
}

// RenderLoading 渲染加載頁面
func RenderLoading(message string) string {
	header := renderSubpageHeader("加載中")
	loadingStyle := lipgloss.NewStyle().Foreground(style.Aurora2)
	loadingText := loadingStyle.Render(fmt.Sprintf("⏳ %s...", message))
	return lipgloss.JoinVertical(lipgloss.Left, header, "", loadingText)
}

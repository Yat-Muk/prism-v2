package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderACMEHTTPInput 渲染 ACME HTTP 输入界面
func RenderACMEHTTPInput(step int, email string, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("申請 ACME 證書 (HTTP-01)")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	var desc, instructions string

	if step == 0 {
		// --- 步驟 1: 輸入郵箱 ---
		desc = lipgloss.NewStyle().
			Foreground(style.Snow2).
			Render(" 步驟 1/2: 設置 ACME 賬戶郵箱")

		instructions = lipgloss.NewStyle().
			Foreground(style.Snow3).
			Render(`
 說明：
  • 這是 ACME 協議 (Let's Encrypt) 要求的賬戶郵箱
  • 用於接收證書過期或安全問題的緊急通知
  • 若不填寫，直接按 Enter 可自動生成隨機郵箱 (不推薦)
  
 請輸入郵箱地址：`)

	} else {
		// --- 步驟 2: 輸入域名 ---
		desc = lipgloss.NewStyle().
			Foreground(style.Snow2).
			Render(" 步驟 2/2: 輸入域名")

		// 顯示上一步確認的郵箱
		emailInfo := lipgloss.NewStyle().
			Foreground(style.StatusGreen).
			Render(fmt.Sprintf(" ✓ 當前賬戶: %s", email))

		note := lipgloss.NewStyle().
			Foreground(style.Snow3).
			Render(`
 注意事項：
  • 需要確保服務器 80 端口處於開放狀態
  • 域名 A 記錄必須正確指向本服務器 IP
  • 申請過程中會臨時佔用 80 端口（約 1-2 分鐘）
  • 適用於單域名證書申請
  
 請輸入要申請證書的域名（例如：example.com）：`)

		instructions = lipgloss.JoinVertical(lipgloss.Left, emailInfo, note)
	}

	statusBlock := RenderStatusMessage(statusMsg)
	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		instructions,
		statusBlock,
		footer,
	)
}

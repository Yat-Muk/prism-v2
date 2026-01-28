package view

import (
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderProviderSwitch 渲染 Provider 切換界面
func RenderProviderSwitch(currentProvider string, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("切換證書提供商")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 選擇 ACME CA 提供商")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// 名稱映射：將後端 ID 轉換為顯示名稱
	providerMap := map[string]string{
		"letsencrypt": "Let's Encrypt",
		"zerossl":     "ZeroSSL",
		"buypass":     "Buypass",
		"google":      "Google Trust Services",
	}

	displayProvider := providerMap[currentProvider]
	// 如果映射中沒有，或者傳入為空，則使用默認邏輯
	if displayProvider == "" {
		if currentProvider != "" {
			displayProvider = currentProvider // 未知提供商，直接顯示 ID
		} else {
			displayProvider = "Let's Encrypt" // 默認值
		}
	}

	// 當前 Provider 顯示樣式
	labelStyle := lipgloss.NewStyle().Foreground(style.Snow2)
	providerStyle := lipgloss.NewStyle().Foreground(style.Aurora4)

	currentLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		labelStyle.Render(" 當前使用: "),
		providerStyle.Render(displayProvider),
	)

	// Provider 列表
	items := []MenuItem{
		{"", "", "", lipgloss.Color("")},
		{constants.KeyProviderType_LetsEncrypt, "Let's Encrypt", "(免費，無需註冊，推薦)", style.StatusGreen},
		{constants.KeyProviderType_ZeroSSL, "ZeroSSL", "(免費，需要 EAB 憑證)", style.Aurora1},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(`
 提示：
  • ZeroSSL 需要註冊 EAB 憑證（腳本將自動處理）
  • 建議首選 Let's Encrypt，兼容性最好`)

	statusBlock := RenderStatusMessage(statusMsg)
	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		currentLine,
		menu,
		instruction,
		statusBlock,
		footer,
	)
}

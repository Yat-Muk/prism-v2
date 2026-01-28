package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderDNSCredentialInput 渲染分步輸入界面
// provider: 當前選擇的提供商
// step: 當前步驟 (0:域名, 1:ID, 2:Secret)
func RenderDNSCredentialInput(provider string, step int, ti textinput.Model, statusMsg string) string {
	// 1. 獲取特定提供商的字段名稱
	idName, secretName := getProviderFieldNames(provider)
	providerName := getProviderDisplayName(provider)

	var (
		stepTitle string // 步驟標題
		stepDesc  string // 頂部描述
		detail    string // 詳細說明區
	)

	// 2. 根據步驟配置文本
	switch step {
	case 0:
		stepTitle = "步驟 1/3: 域名設置"
		stepDesc = " 請輸入要申請證書的域名"
		detail = `
 格式說明：
  • 普通域名：example.com
  • 泛域名：*.example.com
 
 請輸入域名（Esc 返回）：`

	case 1:
		stepTitle = fmt.Sprintf("步驟 2/3: %s 認證", providerName)
		stepDesc = fmt.Sprintf(" 請輸入 %s", idName)
		detail = fmt.Sprintf(`
 憑證獲取提示：
  請登錄 %s 控制台獲取對應憑證。
 
 請輸入 %s（Esc 返回上一步）：`, providerName, idName)

	case 2:
		stepTitle = fmt.Sprintf("步驟 3/3: %s 密鑰", providerName)
		stepDesc = fmt.Sprintf(" 請輸入 %s", secretName)
		detail = fmt.Sprintf(`
 安全提示：
  輸入內容將被隱藏或加密存儲，不會明文顯示。
 
 請輸入 %s（Esc 返回上一步）：`, secretName)
	}

	// 3. 構建界面組件
	header := renderSubpageHeader(fmt.Sprintf("DNS API 申請 - %s", stepTitle))

	descStyle := lipgloss.NewStyle().Foreground(style.Snow2)
	description := descStyle.Render(stepDesc)

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(detail)

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		description,
		divider,
		instruction,
		statusBlock,
		footer,
	)
}

// 輔助：獲取提供商顯示名稱
func getProviderDisplayName(p string) string {
	names := map[string]string{
		"cloudflare":  "Cloudflare",
		"alidns":      "阿里雲 DNS",
		"dnspod":      "騰訊雲 DNSPod",
		"aws":         "AWS Route53",
		"googlecloud": "Google Cloud DNS",
	}
	if name, ok := names[p]; ok {
		return name
	}
	return p
}

// 輔助：獲取不同提供商的字段名稱
func getProviderFieldNames(p string) (idName, secretName string) {
	switch p {
	case "cloudflare":
		return "API 郵箱", "Global API Key"
	case "alidns":
		return "AccessKey ID", "AccessKey Secret"
	case "dnspod":
		return "API ID", "API Token"
	case "aws":
		return "Access Key ID", "Secret Access Key"
	case "googlecloud":
		return "Project ID", "Service Account JSON 路徑"
	default:
		return "API ID", "API Secret"
	}
}

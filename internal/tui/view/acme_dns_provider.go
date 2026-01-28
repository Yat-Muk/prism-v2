package view

import (
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderACMEDNSProvider 渲染 DNS Provider 选择界面
func RenderACMEDNSProvider(ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("申請 ACME 證書 (DNS-01)")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 使用 DNS-01 驗證方式申請證書（支持泛域名）")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	// Provider 列表
	items := []MenuItem{
		{constants.KeyProvider_Cloudflare, "Cloudflare", "(需要 API Email + API Key)", style.Aurora1},
		{constants.KeyProvider_Aliyun, "阿里雲 DNS", "(需要 Access Key + Secret Key)", style.Aurora1},
		{constants.KeyProvider_DNSPod, "騰訊雲 DNSPod", "(需要 API ID + API Token)", style.Aurora1},
		{constants.KeyProvider_AWS, "AWS Route53", "(需要 Access Key + Secret Key)", style.Aurora1},
		{constants.KeyProvider_Google, "Google Cloud DNS", "(需要 Project ID + 服務帳戶文件)", style.Aurora1},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(`
 提示：
  • Cloudflare 支持最完善，推薦使用
  • 申請過程可能需要 1-2 分鐘等待 DNS 生效
  • 憑證將被加密存儲在本地

 按 Esc 返回上級菜單`)

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		menu,
		instruction,
		statusBlock,
		footer,
	)
}

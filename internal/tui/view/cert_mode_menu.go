package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/domain/protocol"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// RenderCertModeMenu 證書模式切換菜單
func RenderCertModeMenu(cfg *config.Config, enabledProtos []int, acmeDomains []string, selfSignedDomains []string, cursor int, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("切換協議證書模式")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 選擇協議證書的來源")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	var supportedProtos []int
	for _, pid := range enabledProtos {
		id := protocol.ID(pid)
		if id == protocol.IDHysteria2 || id == protocol.IDTUIC || id == protocol.IDAnyTLS {
			supportedProtos = append(supportedProtos, pid)
		}
	}

	var items []MenuItem

	if len(supportedProtos) == 0 {
		warnStyle := lipgloss.NewStyle().Foreground(style.StatusYellow)
		snow3Style := lipgloss.NewStyle().Foreground(style.Snow3)
		infoBlock := lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			desc,
			divider,
			"",
			warnStyle.Render(" 提示：當前沒有開啟任何需要本地證書的協議"),
			snow3Style.Render("      (僅支持 Hysteria2 / TUIC / AnyTLS)"),
			snow3Style.Render("      請先去 [配置與協議] -> [協議管理] 中開啟"),
		)
		items = []MenuItem{}
		menu := renderMenuWithAlignment(items, cursor, "", false)
		footer := RenderInputFooter(ti)
		return lipgloss.JoinVertical(lipgloss.Left, infoBlock, menu, footer)
	}

	maxNameWidth := 0
	for _, protoID := range supportedProtos {
		name := getProtocolDisplayName(protoID)
		w := runewidth.StringWidth(name)
		if w > maxNameWidth {
			maxNameWidth = w
		}
	}

	acmeStyle := lipgloss.NewStyle().Foreground(style.StatusGreen)
	selfStyle := lipgloss.NewStyle().Foreground(style.StatusOrange)
	noneStyle := lipgloss.NewStyle().Foreground(style.Snow3)

	for i, protoID := range supportedProtos {
		name := getProtocolDisplayName(protoID)
		certDomain := getProtocolCertDomain(cfg, protoID)
		certMode := getProtocolCertMode(cfg, protoID)

		nameText := name
		w := runewidth.StringWidth(nameText)
		if w < maxNameWidth {
			nameText = nameText + strings.Repeat(" ", maxNameWidth-w)
		}

		var statusText string

		if certMode == "acme" {
			// === ACME 模式 ===
			if certDomain == "" {
				statusText = acmeStyle.Render(" 當前: ACME (未指定域名)")
			} else {
				statusText = acmeStyle.Render(fmt.Sprintf(" 當前: %s (ACME)", certDomain))
			}

		} else if certMode == "self_signed" {
			// === 自簽名模式 ===
			realSelfDomain := "文件未生成"
			if len(selfSignedDomains) > 0 {
				realSelfDomain = selfSignedDomains[0]
			}

			statusText = selfStyle.Render(fmt.Sprintf(" 當前: %s (自簽名)", realSelfDomain))

		} else {
			statusText = noneStyle.Render(" 當前: 未配置")
		}

		items = append(items, MenuItem{
			Num:       fmt.Sprintf("%d", i+1),
			Text:      nameText,
			Desc:      statusText,
			TextColor: style.Snow1,
		})
	}

	menu := renderMenuWithAlignment(items, cursor, "", false)

	instruction := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Render(`
 說明：                                                       
  • 切換 ACME 前，請確保已申請證書                            
  • 切換證書模式後，請同步更新客戶端配置`)

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

func getProtocolDisplayName(protoID int) string {
	switch protocol.ID(protoID) {
	case protocol.IDHysteria2:
		return "Hysteria 2"
	case protocol.IDTUIC:
		return "TUIC v5"
	case protocol.IDAnyTLS:
		return "AnyTLS"
	default:
		return "未知協議"
	}
}

func getProtocolCertDomain(cfg *config.Config, protoID int) string {
	switch protocol.ID(protoID) {
	case protocol.IDHysteria2:
		return cfg.Protocols.Hysteria2.CertDomain
	case protocol.IDTUIC:
		return cfg.Protocols.TUIC.CertDomain
	case protocol.IDAnyTLS:
		return cfg.Protocols.AnyTLS.CertDomain
	default:
		return ""
	}
}

func getProtocolCertMode(cfg *config.Config, protoID int) string {
	switch protocol.ID(protoID) {
	case protocol.IDHysteria2:
		return cfg.Protocols.Hysteria2.CertMode
	case protocol.IDTUIC:
		return cfg.Protocols.TUIC.CertMode
	case protocol.IDAnyTLS:
		return cfg.Protocols.AnyTLS.CertMode
	default:
		return ""
	}
}

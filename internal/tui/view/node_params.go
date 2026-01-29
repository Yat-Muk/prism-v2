package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/charmbracelet/lipgloss"
)

// RenderNodeParams 渲染節點詳細參數
func RenderNodeParams(cfg *config.Config, serverIP string) string {
	var sb strings.Builder
	p := cfg.Protocols

	// --- 1. 樣式定義 ---

	// 標題樣式：紫色背景，帶上下邊距
	titleStyle := lipgloss.NewStyle().
		Foreground(style.Polar1).
		Background(style.Aurora1).
		Padding(0, 1).
		MarginTop(1).
		MarginBottom(0)

	// 鍵樣式：固定寬度 18，右對齊，灰色
	keyStyle := lipgloss.NewStyle().
		Foreground(style.Snow3).
		Width(20).
		Align(lipgloss.Right).
		MarginRight(1)

	// 值樣式：默認亮白色
	valStyle := lipgloss.NewStyle().Foreground(style.Snow1)

	// 高亮樣式：紫色 (用於密碼、UUID、Key)
	highlightStyle := lipgloss.NewStyle().Foreground(style.Aurora3)

	// 狀態樣式
	boolTrueStyle := lipgloss.NewStyle().Foreground(style.StatusGreen) // 綠色
	boolFalseStyle := lipgloss.NewStyle().Foreground(style.StatusRed)  // 紅色

	// --- 2. 輔助渲染函數 ---

	// 渲染普通行
	renderRow := func(key, value string, isHighlight bool) {
		s := valStyle
		if isHighlight {
			s = highlightStyle
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render(key+":"), s.Render(value)))
	}

	// 渲染布爾狀態行 (自定義 True/False 文本)
	renderBoolRow := func(key string, val bool, trueText, falseText string) {
		vStr := falseText
		s := boolFalseStyle
		if val {
			vStr = trueText
			s = boolTrueStyle
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render(key+":"), s.Render(vStr)))
	}

	// ==========================================
	// 1. VLESS Reality Vision
	// ==========================================
	if p.RealityVision.Enabled {
		sb.WriteString(titleStyle.Render("VLESS Reality Vision") + "\n")
		renderRow("Server", serverIP, false)
		renderRow("Server Port", fmt.Sprintf("%d", p.RealityVision.Port), false)
		renderRow("UUID", cfg.UUID, true) // 高亮
		renderRow("Packet Encoding", "xudp", false)
		renderRow("Flow", "xtls-rprx-vision", false)
		renderRow("Network", "tcp", false)
		renderRow("Server Name", p.RealityVision.SNI, false)
		renderRow("Fingerprint", "chrome", false)
		renderRow("Public Key", p.RealityVision.PublicKey, true) // 高亮
		renderRow("Short ID", p.RealityVision.ShortID, true)     // 高亮
	}

	// ==========================================
	// 2. VLESS Reality gRPC
	// ==========================================
	if p.RealityGRPC.Enabled {
		sb.WriteString(titleStyle.Render("VLESS Reality gRPC") + "\n")
		renderRow("Server", serverIP, false)
		renderRow("Server Port", fmt.Sprintf("%d", p.RealityGRPC.Port), false)
		renderRow("UUID", cfg.UUID, true)
		renderRow("Network", "grpc", false)
		renderRow("Service Name", "grpc", false)
		renderRow("Server Name", p.RealityGRPC.SNI, false)
		renderRow("Public Key", p.RealityGRPC.PublicKey, true)
		renderRow("Short ID", p.RealityGRPC.ShortID, true)
	}

	// ==========================================
	// 3. Hysteria 2
	// ==========================================
	if p.Hysteria2.Enabled {
		sb.WriteString(titleStyle.Render("Hysteria 2") + "\n")

		addr := serverIP
		if p.Hysteria2.CertMode == "acme" && p.Hysteria2.CertDomain != "" {
			addr = p.Hysteria2.CertDomain
		}

		renderRow("Server", addr, false)
		renderRow("Server Port", fmt.Sprintf("%d", p.Hysteria2.Port), false)

		if p.Hysteria2.PortHopping != "" {
			renderRow("Port Hopping", p.Hysteria2.PortHopping, false)
		}

		pass := p.Hysteria2.Password
		if pass == "" {
			pass = cfg.Password
		}
		renderRow("Password", pass, true)

		renderRow("Server Name", p.Hysteria2.SNI, false)
		renderRow("ALPN", p.Hysteria2.ALPN, false)

		// 智能判斷是否不安全
		isInsecure := (p.Hysteria2.CertMode == "self_signed")
		renderBoolRow("Insecure", isInsecure, "true", "false")

		if p.Hysteria2.Obfs != "" {
			renderRow("Obfs Type", "salamander", false)
			renderRow("Obfs Password", p.Hysteria2.Obfs, true)
		}

		bw := fmt.Sprintf("%d Mbps (Up) / %d Mbps (Down)", p.Hysteria2.UpMbps, p.Hysteria2.DownMbps)
		renderRow("Bandwidth", bw, false)
	}

	// ==========================================
	// 4. TUIC v5
	// ==========================================
	if p.TUIC.Enabled {
		sb.WriteString(titleStyle.Render("TUIC v5") + "\n")

		addr := serverIP
		if p.TUIC.CertMode == "acme" && p.TUIC.CertDomain != "" {
			addr = p.TUIC.CertDomain
		}

		renderRow("Server", addr, false)
		renderRow("Server Port", fmt.Sprintf("%d", p.TUIC.Port), false)

		uuid := p.TUIC.UUID
		if uuid == "" {
			uuid = cfg.UUID
		}
		renderRow("UUID", uuid, true)

		pass := p.TUIC.Password
		if pass == "" {
			pass = cfg.Password
		}
		renderRow("Password", pass, true)

		renderRow("Server Name", p.TUIC.SNI, false)
		renderRow("Congestion Control", p.TUIC.CongestionControl, false)
		renderRow("ALPN", "h3", false)
		renderRow("UDP Relay Mode", "native", false)

		isInsecure := (p.TUIC.CertMode == "self_signed")
		renderBoolRow("Insecure", isInsecure, "true", "false")
	}

	// ==========================================
	// 5. AnyTLS (HTTP/2)
	// ==========================================
	if p.AnyTLS.Enabled {
		sb.WriteString(titleStyle.Render("AnyTLS") + "\n")

		addr := serverIP
		if p.AnyTLS.CertMode == "acme" && p.AnyTLS.CertDomain != "" {
			addr = p.AnyTLS.CertDomain
		}

		renderRow("Server", addr, false)
		renderRow("Server Port", fmt.Sprintf("%d", p.AnyTLS.Port), false)
		renderRow("Username", p.AnyTLS.Username, false)

		pass := p.AnyTLS.Password
		if pass == "" {
			pass = cfg.Password
		}
		renderRow("Password", pass, true)

		renderRow("Server Name", p.AnyTLS.SNI, false)
		renderRow("ALPN", "h2, http/1.1", false)

		isInsecure := (p.AnyTLS.CertMode == "self_signed")
		renderBoolRow("Insecure", isInsecure, "true", "false")
	}

	// ==========================================
	// 6. AnyTLS Reality
	// ==========================================
	if p.AnyTLSReality.Enabled {
		sb.WriteString(titleStyle.Render("AnyTLS Reality") + "\n")
		renderRow("Server", serverIP, false)
		renderRow("Server Port", fmt.Sprintf("%d", p.AnyTLSReality.Port), false)
		renderRow("Username", p.AnyTLSReality.Username, false)

		pass := p.AnyTLSReality.Password
		if pass == "" {
			pass = cfg.Password
		}
		renderRow("Password", pass, true)

		// 補全了你之前缺失的參數
		renderRow("Server Name", p.AnyTLSReality.SNI, false)
		renderRow("Fingerprint", "chrome", false)

		// Reality 核心參數
		renderRow("Public Key", p.AnyTLSReality.PublicKey, true) // 高亮
		renderRow("Short ID", p.AnyTLSReality.ShortID, true)     // 高亮
	}

	// ==========================================
	// 7. ShadowTLS v3
	// ==========================================
	if p.ShadowTLS.Enabled {
		sb.WriteString(titleStyle.Render("ShadowTLS v3") + "\n")
		renderRow("Server", serverIP, false)
		renderRow("Server Port", fmt.Sprintf("%d", p.ShadowTLS.Port), false)

		// 握手密碼
		tlsPass := p.ShadowTLS.Password
		if tlsPass == "" {
			tlsPass = cfg.Password
		}
		renderRow("Handshake Pwd", tlsPass, true)

		renderRow("Server Name", p.ShadowTLS.SNI, false)
		renderRow("Version", "3", false)
		renderBoolRow("Strict Mode", p.ShadowTLS.StrictMode, "On", "Off")

		// 底層 SS 分隔
		sb.WriteString("\n")
		renderRow("SS Cipher", p.ShadowTLS.SSMethod, false)

		ssPass := p.ShadowTLS.SSPassword
		if ssPass == "" {
			ssPass = cfg.Password
		}
		renderRow("SS Password", ssPass, true)
	}

	return sb.String()
}

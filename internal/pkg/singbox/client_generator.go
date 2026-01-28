package singbox

import (
	"github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/domain/protocol"
)

// GenerateClientConfig 使用協議工廠生成客戶端配置 (v1.12+ 規範)
func GenerateClientConfig(serverCfg *config.Config, defaultHost string, factory protocol.Factory) *Config {
	protocols := factory.FromConfig(serverCfg)

	var proxyTags []string
	var outbounds []Outbound

	for _, p := range protocols {
		if !p.IsEnabled() {
			continue
		}

		out, err := p.ToSingboxOutbound()
		if err != nil {
			continue
		}

		finalAddress := defaultHost

		switch p.Type() {
		case protocol.TypeHysteria2:
			if serverCfg.Protocols.Hysteria2.CertMode == "acme" && serverCfg.Protocols.Hysteria2.CertDomain != "" {
				finalAddress = serverCfg.Protocols.Hysteria2.CertDomain
			}
		case protocol.TypeTUIC:
			if serverCfg.Protocols.TUIC.CertMode == "acme" && serverCfg.Protocols.TUIC.CertDomain != "" {
				finalAddress = serverCfg.Protocols.TUIC.CertDomain
			}
		case protocol.TypeAnyTLS:
			if serverCfg.Protocols.AnyTLS.CertMode == "acme" && serverCfg.Protocols.AnyTLS.CertDomain != "" {
				finalAddress = serverCfg.Protocols.AnyTLS.CertDomain
			}
		}

		// 應用地址
		if _, ok := out["server"]; ok {
			out["server"] = finalAddress
		}

		// 使用協議名稱作為 Tag
		tag := p.Name()
		out["tag"] = tag
		proxyTags = append(proxyTags, tag)
		outbounds = append(outbounds, out)

		// 特殊處理 ShadowTLS 的 Detour
		if s, ok := p.(interface{ GetDetourOutbound() map[string]interface{} }); ok {
			detourOut := s.GetDetourOutbound()
			if _, ok := detourOut["server"]; ok {
				detourOut["server"] = finalAddress
			}
			outbounds = append(outbounds, detourOut)
		}
	}

	// 3. 構建 Selector (節點選擇器)
	if len(proxyTags) > 0 {
		// URLTest 自動選擇
		outbounds = append([]Outbound{{
			"type":      "urltest",
			"tag":       "⚡️ Auto",
			"outbounds": proxyTags,
			"url":       "https://www.gstatic.com/generate_204",
			"interval":  "3m",
		}}, outbounds...)

		// 主選擇器
		selectorTags := append([]string{"⚡️ Auto"}, proxyTags...)
		selector := Outbound{
			"type":      "selector",
			"tag":       "🚀 Proxy",
			"outbounds": selectorTags,
			"default":   "⚡️ Auto",
		}
		outbounds = append([]Outbound{selector}, outbounds...)
	} else {
		// 防禦性代碼：無節點時
		outbounds = append(outbounds, Outbound{"type": "block", "tag": "🚀 Proxy"})
	}

	// 4. 添加基礎組件
	outbounds = append(outbounds,
		Outbound{"type": "direct", "tag": "direct"},
	)

	// 5. 構建客戶端 Inbounds (標準 TUN + Mixed)
	inbounds := []Inbound{
		{
			"type":                       "tun",
			"tag":                        "tun-in",
			"interface_name":             "tun0",
			"inet4_address":              "172.19.0.1/30",
			"auto_route":                 true,
			"strict_route":               true,
			"stack":                      "mixed",
			"sniff":                      true,
			"sniff_override_destination": false,
		},
		{
			"type":        "mixed",
			"tag":         "mixed-in",
			"listen":      "127.0.0.1",
			"listen_port": 2333,
			"sniff":       true,
		},
	}

	// 6. 構建 DNS
	dns := &DNS{
		Servers: []DNSServer{
			{
				Tag:    "remote",
				Server: "8.8.8.8",
				Type:   "udp",
				Detour: "🚀 Proxy",
			},
			{
				Tag:    "local",
				Type:   "local",
				Detour: "direct",
			},
		},
		Rules: []DNSRule{
			{Outbound: "any", Server: "local"}, // 攔截 DNS 洩漏
			{RuleSet: []string{"geosite-cn"}, Server: "local"},
		},
		Final: "remote",
	}

	// 7. 構建路由規則
	route := &Route{
		RuleSet: []RuleSet{
			{
				Tag:            "geosite-cn",
				Type:           "remote",
				Format:         "binary",
				URL:            "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs",
				DownloadDetour: "🚀 Proxy",
			},
			{
				Tag:            "geoip-cn",
				Type:           "remote",
				Format:         "binary",
				URL:            "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs",
				DownloadDetour: "🚀 Proxy",
			},
			{
				Tag:            "geosite-category-ads-all",
				Type:           "remote",
				Format:         "binary",
				URL:            "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-ads-all.srs",
				DownloadDetour: "🚀 Proxy",
			},
		},
		Rules: []RouteRule{
			{Protocol: "dns", Action: "hijack-dns"},
			{RuleSet: []string{"geosite-category-ads-all"}, Action: "reject"},
			{RuleSet: []string{"geoip-cn", "geosite-cn"}, Outbound: "direct"},
		},
		Final:               "🚀 Proxy",
		AutoDetectInterface: true,
		DefaultDomainResolver: map[string]any{
			"server": "remote",
		},
	}

	return &Config{
		Log:       &Log{Level: "info", Timestamp: true},
		DNS:       dns,
		Inbounds:  inbounds,
		Outbounds: outbounds,
		Route:     route,
	}
}

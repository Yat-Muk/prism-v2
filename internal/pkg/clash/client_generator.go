package clash

import (
	"github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/domain/protocol"
)

// Config Clash 配置文件結構
type Config struct {
	Port               int                      `yaml:"port"`
	SocksPort          int                      `yaml:"socks-port"`
	RedirPort          int                      `yaml:"redir-port"`
	MixedPort          int                      `yaml:"mixed-port"`
	AllowLan           bool                     `yaml:"allow-lan"`
	Mode               string                   `yaml:"mode"`
	LogLevel           string                   `yaml:"log-level"`
	ExternalController string                   `yaml:"external-controller"`
	DNS                DNSConfig                `yaml:"dns"`
	Proxies            []map[string]interface{} `yaml:"proxies"`
	ProxyGroups        []map[string]interface{} `yaml:"proxy-groups"`
	Rules              []string                 `yaml:"rules"`
}

type DNSConfig struct {
	Enable     bool     `yaml:"enable"`
	Listen     string   `yaml:"listen"`
	Default    []string `yaml:"default-nameserver"`
	Enhanced   string   `yaml:"enhanced-mode"`
	FakeIP     FakeIP   `yaml:"fake-ip-range"`
	UseHosts   bool     `yaml:"use-hosts"`
	Nameserver []string `yaml:"nameserver"`
	Fallback   []string `yaml:"fallback"`
}

type FakeIP struct {
	Ranges []string `yaml:"ip-cidr"`
}

// GenerateClientConfig 生成 Clash Meta 配置
func GenerateClientConfig(serverCfg *config.Config, defaultHost string, factory protocol.Factory) *Config {
	protocols := factory.FromConfig(serverCfg)
	var proxies []map[string]interface{}
	var proxyNames []string

	// 1. 遍歷協議轉換為 Clash Proxies
	for _, p := range protocols {
		if !p.IsEnabled() {
			continue
		}

		proxy := protocolToClashProxy(p, defaultHost, serverCfg)

		if proxy != nil {
			proxies = append(proxies, proxy)
			if name, ok := proxy["name"].(string); ok {
				proxyNames = append(proxyNames, name)
			}
		}
	}

	// 2. 構建 Proxy Groups
	// 自動選擇
	autoGroup := map[string]interface{}{
		"name":      "⚡️ Auto",
		"type":      "url-test",
		"proxies":   proxyNames,
		"url":       "https://www.gstatic.com/generate_204",
		"interval":  300,
		"tolerance": 50,
	}

	// 主選擇器
	selectorNames := append([]string{"⚡️ Auto"}, proxyNames...)
	mainGroup := map[string]interface{}{
		"name":    "🚀 Proxy",
		"type":    "select",
		"proxies": selectorNames,
	}

	groups := []map[string]interface{}{mainGroup, autoGroup}

	// 3. 構建規則
	rules := []string{
		"GEOIP,CN,DIRECT",
		"MATCH,🚀 Proxy",
	}

	// 4. 返回完整配置
	return &Config{
		Port:      7890,
		SocksPort: 7891,
		MixedPort: 7893,
		AllowLan:  false,
		Mode:      "rule",
		LogLevel:  "info",
		DNS: DNSConfig{
			Enable:   true,
			Listen:   "0.0.0.0:1053",
			Enhanced: "fake-ip",
			FakeIP:   FakeIP{Ranges: []string{"198.18.0.1/16"}},
			Default:  []string{"223.5.5.5", "119.29.29.29"},
			Nameserver: []string{
				"https://dns.alidns.com/dns-query",
				"https://doh.pub/dns-query",
			},
			Fallback: []string{
				"https://1.1.1.1/dns-query",
				"https://8.8.8.8/dns-query",
			},
		},
		Proxies:     proxies,
		ProxyGroups: groups,
		Rules:       rules,
	}
}

// protocolToClashProxy 將內部協議對象轉換為 Clash Meta 代理 Map
func protocolToClashProxy(p protocol.Protocol, defaultHost string, cfg *config.Config) map[string]interface{} {
	// [核心修復] 確定地址邏輯
	finalAddress := defaultHost

	switch p.Type() {
	case protocol.TypeHysteria2:
		if cfg.Protocols.Hysteria2.CertMode == "acme" && cfg.Protocols.Hysteria2.CertDomain != "" {
			finalAddress = cfg.Protocols.Hysteria2.CertDomain
		}
	case protocol.TypeTUIC:
		if cfg.Protocols.TUIC.CertMode == "acme" && cfg.Protocols.TUIC.CertDomain != "" {
			finalAddress = cfg.Protocols.TUIC.CertDomain
		}
	case protocol.TypeAnyTLS:
		if cfg.Protocols.AnyTLS.CertMode == "acme" && cfg.Protocols.AnyTLS.CertDomain != "" {
			finalAddress = cfg.Protocols.AnyTLS.CertDomain
		}
	}

	base := map[string]interface{}{
		"name":   p.Name(),
		"server": finalAddress,
		"port":   p.Port(),
	}

	switch v := p.(type) {
	case *protocol.RealityVision:
		base["type"] = "vless"
		base["uuid"] = v.Users[0].UUID
		base["network"] = "tcp"
		base["tls"] = true
		base["udp"] = true
		base["flow"] = "xtls-rprx-vision"
		base["servername"] = v.SNI
		base["client-fingerprint"] = "chrome"
		base["reality-opts"] = map[string]interface{}{
			"public-key": v.PublicKey,
			"short-id":   v.ShortID,
		}

	case *protocol.RealityGRPC:
		base["type"] = "vless"
		base["uuid"] = v.Users[0].UUID
		base["network"] = "grpc"
		base["tls"] = true
		base["udp"] = true
		base["servername"] = v.SNI
		base["client-fingerprint"] = "chrome"
		base["grpc-opts"] = map[string]interface{}{
			"grpc-service-name": v.ServiceName,
		}
		base["reality-opts"] = map[string]interface{}{
			"public-key": v.PublicKey,
			"short-id":   v.ShortID,
		}

	case *protocol.Hysteria2:
		base["type"] = "hysteria2"
		base["password"] = v.Password
		base["sni"] = v.SNI
		base["skip-cert-verify"] = true
		base["alpn"] = []string{v.ALPN}
		if v.Obfs != "" {
			base["obfs"] = "salamander"
			base["obfs-password"] = v.Obfs
		}

	case *protocol.TUIC:
		base["type"] = "tuic"
		base["uuid"] = v.UUID
		base["password"] = v.Password
		base["sni"] = v.SNI
		base["skip-cert-verify"] = true
		base["alpn"] = v.ALPN
		base["congestion-controller"] = v.CongestionControl
		base["udp-relay-mode"] = "native"

	case *protocol.AnyTLS:
		// AnyTLS 本質是 HTTP/2 代理
		base["type"] = "http"
		base["username"] = v.Username
		base["password"] = v.Password
		base["tls"] = true
		base["sni"] = v.SNI
		base["skip-cert-verify"] = true

	case *protocol.ShadowTLS:
		// Clash Meta 支持 shadow-tls 作為 SS 的插件
		base["type"] = "ss"
		base["cipher"] = v.SSMethod
		base["password"] = v.SSPassword
		base["plugin"] = "shadow-tls"
		base["plugin-opts"] = map[string]interface{}{
			"host":     v.SNI,
			"password": v.Password,
			"version":  3,
		}

	default:
		return nil
	}

	return base
}

package singbox

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/domain/protocol"
	"github.com/Yat-Muk/prism-v2/internal/pkg/errors"
	"github.com/go-acme/lego/v4/log"
)

type generator struct {
	protocolFactory protocol.Factory
	version         string
	mu              sync.Mutex // 加鎖防止並發讀寫
}

func NewGenerator(version string, protocolFactory protocol.Factory) Generator {
	return &generator{
		protocolFactory: protocolFactory,
		version:         strings.TrimSpace(version),
	}
}

// detectVersion 嘗試動態獲取 sing-box 版本
func (g *generator) detectVersion() {
	g.mu.Lock()
	defer g.mu.Unlock()

	// 如果已經有有效版本號，跳過檢測 (避免頻繁調用外部命令)
	if g.version != "" && g.version != "unknown" {
		return
	}

	// 設置 2 秒超時，避免長時間阻塞
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 執行命令獲取版本
	cmd := exec.CommandContext(ctx, "sing-box", "version")
	out, err := cmd.Output()
	if err == nil {
		// 輸出示例: "sing-box version 1.12.13 ..."
		parts := strings.Fields(string(out))
		if len(parts) >= 3 && parts[1] == "version" {
			g.version = parts[2]
		}
	} else {
		// 記錄警告但不中斷流程，可能用戶未安裝 sing-box
		if ctx.Err() == context.DeadlineExceeded {
			log.Warnf("檢測 sing-box 版本超時")
		}
	}
}

func (g *generator) isLegacyCore() bool {
	g.mu.Lock()
	v := g.version
	g.mu.Unlock()

	v = strings.TrimPrefix(v, "v")

	// 如果版本未知，默認視為新版 (False)
	// 因為新版 sing-box 對舊配置是 Fatal 錯誤，而舊版對新配置通常只是忽略或 Warn
	// 且現在大多數用戶安裝的都是 1.12+
	if v == "" {
		return false
	}

	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return false // 格式不對也視為新版，安全起見
	}

	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])

	// 定義舊版：主版本為1 且 次版本 < 12 (即 1.11.x 及以下)
	return major == 1 && minor < 12
}

func (g *generator) isV112Plus() bool {
	return !g.isLegacyCore()
}

func (g *generator) generateInboundsFromProtocols(protocols []protocol.Protocol) []Inbound {
	var inbounds []Inbound

	for _, proto := range protocols {
		// 特殊處理 ShadowTLS：需要先添加 Shadowsocks detour 入站
		if shadowtls, ok := proto.(*protocol.ShadowTLS); ok {
			// 1. 先添加 Shadowsocks detour 入站
			detourInbound := shadowtls.GetDetourInbound()
			inbounds = append(inbounds, Inbound(detourInbound))

			// 2. 再添加 ShadowTLS 主入站
			if inboundMap, err := proto.ToSingboxInbound(); err == nil {
				inbounds = append(inbounds, Inbound(inboundMap))
			} else {
				log.Warnf("跳過協議 %s (端口 %d): %v",
					proto.Name(), proto.Port(), err)
			}
			continue
		}

		// 其他協議正常處理
		if inboundMap, err := proto.ToSingboxInbound(); err == nil {
			inbounds = append(inbounds, Inbound(inboundMap))
		} else {
			log.Warnf("跳過協議 %s (端口 %d): %v",
				proto.Name(), proto.Port(), err)
		}
	}

	return inbounds
}

func (g *generator) Generate(ctx context.Context, cfg *domainConfig.Config) (*Config, error) {
	if cfg == nil {
		return nil, errors.New("SINGBOX001", "配置不能為空")
	}

	// 每次生成前，嘗試自動檢測一次版本
	// 這確保了即使初始化時版本為空，生成時也能獲取到正確版本
	g.detectVersion()

	log := g.generateLog(cfg)

	dns, err := g.GenerateDNS(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "SINGBOX002", "生成 DNS 配置失敗")
	}

	protocols := g.protocolFactory.FromConfig(cfg)
	inbounds := g.generateInboundsFromProtocols(protocols)

	outbounds, err := g.GenerateOutbounds(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "SINGBOX004", "生成 outbound 配置失敗")
	}

	route, err := g.GenerateRoute(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "SINGBOX005", "生成路由配置失敗")
	}

	return &Config{
		Log:       log,
		DNS:       dns,
		Inbounds:  inbounds,
		Outbounds: outbounds,
		Route:     route,
	}, nil
}

func (g *generator) generateLog(cfg *domainConfig.Config) *Log {
	return &Log{
		Level:     cfg.Log.Level,
		Timestamp: true,
	}
}

func (g *generator) GenerateDNS(ctx context.Context, cfg *domainConfig.Config) (*DNS, error) {
	if g.isLegacyCore() {
		// 舊版 (< 1.12) 配置
		strategy := cfg.Routing.DomainStrategy
		if strategy == "" {
			strategy = "prefer_ipv4"
		}

		servers := []DNSServer{
			{Tag: "dns_google", Address: "8.8.8.8"},
			{Tag: "dns_local", Address: "local"},
		}

		rules := []DNSRule{{RuleSet: []string{"geosite-cn"}, Server: "dns_local"}}
		return &DNS{Servers: servers, Rules: rules, Final: "dns_google", Strategy: strategy}, nil
	}

	// 新版 (1.12+) 配置：無 Strategy 字段，Type 字段更明確
	servers := []DNSServer{
		{Tag: "dns_google", Server: "8.8.8.8", Type: "udp"},
		{Tag: "dns_local", Type: "local"},
	}

	rules := []DNSRule{{RuleSet: []string{"geosite-cn"}, Server: "dns_local"}}
	return &DNS{Servers: servers, Rules: rules, Final: "dns_google"}, nil
}

func (g *generator) GenerateInbounds(ctx context.Context, protocols []protocol.Protocol) ([]Inbound, error) {
	return g.generateInboundsFromProtocols(protocols), nil
}

func (g *generator) GenerateOutbounds(ctx context.Context, cfg *domainConfig.Config) ([]Outbound, error) {
	var outbounds []Outbound

	if g.isLegacyCore() {
		// 舊版 (< 1.12) 包含 block 和 dns-out
		outbounds = append(outbounds,
			Outbound{"type": "direct", "tag": "direct"},
			Outbound{"type": "block", "tag": "block"},
			Outbound{"type": "dns", "tag": "dns-out"},
		)
	} else {
		// 新版 (>= 1.12) 移除了 block 和 dns 類型的 outbound
		outbounds = append(outbounds, Outbound{"type": "direct", "tag": "direct"})
	}

	if cfg.Routing.IPv6Split.Enabled {
		o := Outbound{"type": "direct", "tag": "ipv6-out"}
		if g.isLegacyCore() {
			o["domain_strategy"] = "ipv6_only"
		}
		// 新版移除 domain_strategy
		outbounds = append(outbounds, o)
	}

	if cfg.Routing.WARP.Enabled {
		outbounds = append(outbounds, g.generateWARPOutbound(cfg))
	}

	return outbounds, nil
}

func (g *generator) generateWARPOutbound(cfg *domainConfig.Config) Outbound {
	localAddr := []string{"172.16.0.2/32"}
	if cfg.Routing.WARP.IPv6 != "" {
		localAddr = append(localAddr, cfg.Routing.WARP.IPv6+"/128")
	}

	return Outbound{
		"type":            "wireguard",
		"tag":             "warp-out",
		"server":          "162.159.192.1",
		"server_port":     2408,
		"local_address":   localAddr,
		"private_key":     cfg.Routing.WARP.PrivateKey,
		"peer_public_key": "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo=",
		"mtu":             1280,
	}
}

func (g *generator) GenerateRoute(ctx context.Context, cfg *domainConfig.Config) (*Route, error) {
	ruleSets := []RuleSet{
		{
			Tag:            "geosite-cn",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs",
			DownloadDetour: "direct",
		},
		{
			Tag:            "geoip-cn",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs",
			DownloadDetour: "direct",
		},
		{
			Tag:            "geosite-ads",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-ads-all.srs",
			DownloadDetour: "direct",
		},
	}

	var rules []RouteRule

	if g.isLegacyCore() {
		// 舊版 (< 1.12) 使用 Outbound
		rules = append(rules,
			RouteRule{Protocol: "dns", Outbound: "dns-out"},
			RouteRule{RuleSet: []string{"geosite-ads"}, Outbound: "block"},
		)
	} else {
		// 新版 (>= 1.12) 使用 Action
		rules = append(rules,
			RouteRule{Protocol: "dns", Action: "hijack-dns"},
			RouteRule{RuleSet: []string{"geosite-ads"}, Action: "reject"},
		)
	}

	if cfg.Routing.WARP.Enabled && len(cfg.Routing.WARP.Domains) > 0 {
		rules = append(rules, RouteRule{Domain: cfg.Routing.WARP.Domains, Outbound: "warp-out"})
	}

	if cfg.Routing.IPv6Split.Enabled && len(cfg.Routing.IPv6Split.Domains) > 0 {
		rules = append(rules, RouteRule{Domain: cfg.Routing.IPv6Split.Domains, Outbound: "ipv6-out"})
	}

	rules = append(rules, RouteRule{RuleSet: []string{"geoip-cn", "geosite-cn"}, Outbound: "direct"})

	route := &Route{
		Rules:               rules,
		RuleSet:             ruleSets,
		Final:               "direct",
		AutoDetectInterface: true,
	}

	if g.isV112Plus() {
		route.DefaultDomainResolver = map[string]any{"server": "dns_google"}
	}

	return route, nil
}

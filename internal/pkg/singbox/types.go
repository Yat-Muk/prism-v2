package singbox

// Config Sing-box 完整配置結構
// 參考文檔: https://sing-box.sagernet.org/configuration/
type Config struct {
	Log       *Log       `json:"log,omitempty"`
	DNS       *DNS       `json:"dns,omitempty"`
	Inbounds  []Inbound  `json:"inbounds,omitempty"`
	Outbounds []Outbound `json:"outbounds,omitempty"`
	Route     *Route     `json:"route,omitempty"`
	// Experimental 字段可根據需要添加
	Experimental map[string]interface{} `json:"experimental,omitempty"`
}

// Log 日誌配置
type Log struct {
	Level     string `json:"level,omitempty"`
	Output    string `json:"output,omitempty"`
	Timestamp bool   `json:"timestamp,omitempty"`
}

// DNS 配置
type DNS struct {
	Servers          []DNSServer `json:"servers,omitempty"`
	Rules            []DNSRule   `json:"rules,omitempty"`
	Final            string      `json:"final,omitempty"`
	Strategy         string      `json:"strategy,omitempty"` // 舊版兼容 (ipv4_only 等)
	DisableCache     bool        `json:"disable_cache,omitempty"`
	DisableExpire    bool        `json:"disable_expire,omitempty"`
	IndependentCache bool        `json:"independent_cache,omitempty"`
	ReverseMapping   bool        `json:"reverse_mapping,omitempty"`
	FakeIP           *FakeIP     `json:"fakeip,omitempty"`
}

// FakeIP 配置
type FakeIP struct {
	Enabled    bool   `json:"enabled,omitempty"`
	Inet4Range string `json:"inet4_range,omitempty"`
	Inet6Range string `json:"inet6_range,omitempty"`
}

// DNSServer DNS 服務器
type DNSServer struct {
	Tag     string `json:"tag,omitempty"`
	Address string `json:"address,omitempty"` // 兼容舊版 address 字段
	Server  string `json:"server,omitempty"`  // 實際服務器地址 (v1.12+ 推薦)
	Type    string `json:"type,omitempty"`    // v1.12+ (udp, tcp, tls, https, quic, local...)
	Detour  string `json:"detour,omitempty"`

	// 特定類型字段
	ClientSubnet string `json:"client_subnet,omitempty"`
}

// DNSRule DNS 規則
type DNSRule struct {
	RuleSet      []string `json:"rule_set,omitempty"` // 匹配規則集
	Domain       []string `json:"domain,omitempty"`   // 匹配域名
	DomainSuffix []string `json:"domain_suffix,omitempty"`

	Server string `json:"server,omitempty"` // 指定 DNS 服務器 Tag

	// 動作字段
	Outbound string `json:"outbound,omitempty"` // 舊版行為
	Action   string `json:"action,omitempty"`   // v1.12+ (route, reject, hijack-dns)
	Method   string `json:"method,omitempty"`   // 伴隨 Action 使用

	// 其他匹配條件
	Inbound   []string `json:"inbound,omitempty"`
	IPVersion int      `json:"ip_version,omitempty"`
	AuthUser  []string `json:"auth_user,omitempty"`
	Protocol  []string `json:"protocol,omitempty"`
}

// Inbound 入站配置 (通用 Map)
// 由於 Inbound 類型繁多 (tun, mixed, socks, vless...) 且字段差異巨大，
// 使用 map[string]interface{} 是最靈活且標準的做法。
type Inbound map[string]interface{}

// Outbound 出站配置 (通用 Map)
// 同上，Outbound 包含 vless, hysteria2, shadowtls, selector, urltest 等多種類型。
type Outbound map[string]interface{}

// Route 路由配置
type Route struct {
	Rules               []RouteRule `json:"rules,omitempty"`
	RuleSet             []RuleSet   `json:"rule_set,omitempty"`
	Final               string      `json:"final,omitempty"`
	AutoDetectInterface bool        `json:"auto_detect_interface,omitempty"`
	OverrideAndroidVPN  bool        `json:"override_android_vpn,omitempty"`
	DefaultInterface    string      `json:"default_interface,omitempty"`
	DefaultMark         int         `json:"default_mark,omitempty"`

	// v1.12+ 新增字段
	DefaultDomainResolver map[string]any `json:"default_domain_resolver,omitempty"`
}

// RouteRule 路由規則
type RouteRule struct {
	// Protocol 在 Sing-box 規範中是字符串數組 (例如 ["dns", "quic"])。
	// 為了兼容代碼中可能出現的 {Protocol: "dns"} 賦值寫法，這裡定義為 interface{}。
	// 在 JSON 序列化時，如果是字符串會自動轉為字符串，如果是數組則轉為數組。
	Protocol interface{} `json:"protocol,omitempty"`

	RuleSet       []string `json:"rule_set,omitempty"`
	Domain        []string `json:"domain,omitempty"`
	DomainSuffix  []string `json:"domain_suffix,omitempty"`
	DomainKeyword []string `json:"domain_keyword,omitempty"`
	DomainRegex   []string `json:"domain_regex,omitempty"`

	IPCIDR      []string `json:"ip_cidr,omitempty"`
	IPIsPrivate bool     `json:"ip_is_private,omitempty"`

	SourceIPCIDR []string `json:"source_ip_cidr,omitempty"`
	SourcePort   []int    `json:"source_port,omitempty"`
	Port         []int    `json:"port,omitempty"`

	Network []string `json:"network,omitempty"` // tcp, udp
	Inbound []string `json:"inbound,omitempty"`

	// 動作字段
	Action   string `json:"action,omitempty"`   // v1.12+ (route, route-options, reject, hijack-dns, sniff)
	Outbound string `json:"outbound,omitempty"` // 舊版 (相當於 action: route)
}

// RuleSet 規則集資源
type RuleSet struct {
	Tag            string `json:"tag"`
	Type           string `json:"type"`           // local, remote
	Format         string `json:"format"`         // source, binary
	URL            string `json:"url,omitempty"`  // remote 專用
	Path           string `json:"path,omitempty"` // local 專用
	DownloadDetour string `json:"download_detour,omitempty"`
	UpdateInterval string `json:"update_interval,omitempty"`
}

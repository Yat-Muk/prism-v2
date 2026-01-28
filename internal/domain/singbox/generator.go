package singbox

import (
	"context"

	"github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/domain/protocol"
)

// Generator Sing-box 配置生成器接口
type Generator interface {
	Generate(ctx context.Context, cfg *config.Config) (*Config, error)
	GenerateInbounds(ctx context.Context, protocols []protocol.Protocol) ([]Inbound, error)
	GenerateOutbounds(ctx context.Context, cfg *config.Config) ([]Outbound, error)
	GenerateRoute(ctx context.Context, cfg *config.Config) (*Route, error)
	GenerateDNS(ctx context.Context, cfg *config.Config) (*DNS, error)
}

// Config Sing-box 完整配置
type Config struct {
	Log       *Log       `json:"log"`
	DNS       *DNS       `json:"dns"`
	Inbounds  []Inbound  `json:"inbounds"`
	Outbounds []Outbound `json:"outbounds"`
	Route     *Route     `json:"route"`
}

type Log struct {
	Level     string `json:"level"`
	Timestamp bool   `json:"timestamp"`
}

type DNS struct {
	Servers  []DNSServer `json:"servers"`
	Rules    []DNSRule   `json:"rules,omitempty"`
	Final    string      `json:"final,omitempty"`
	Strategy string      `json:"strategy,omitempty"`
}

type DNSServer struct {
	Tag     string `json:"tag"`
	Server  string `json:"server,omitempty"`
	Address string `json:"address,omitempty"`
	Type    string `json:"type,omitempty"`
	Detour  string `json:"detour,omitempty"`
}

type DNSRule struct {
	RuleSet  []string `json:"rule_set,omitempty"`
	Domain   []string `json:"domain,omitempty"`
	Server   string   `json:"server,omitempty"`
	Outbound string   `json:"outbound,omitempty"`
}

type Inbound map[string]interface{}
type Outbound map[string]interface{}

type Route struct {
	Rules               []RouteRule `json:"rules,omitempty"`
	RuleSet             []RuleSet   `json:"rule_set,omitempty"`
	Final               string      `json:"final,omitempty"`
	AutoDetectInterface bool        `json:"auto_detect_interface"`
	// 1.12+ 專用，可選
	DefaultDomainResolver map[string]any `json:"default_domain_resolver,omitempty"`
}

type RouteRule struct {
	Protocol string   `json:"protocol,omitempty"`
	RuleSet  []string `json:"rule_set,omitempty"`
	Domain   []string `json:"domain,omitempty"`
	Action   string   `json:"action,omitempty"`
	Outbound string   `json:"outbound,omitempty"`
}

type RuleSet struct {
	Tag            string `json:"tag"`
	Type           string `json:"type"`
	Format         string `json:"format"`
	URL            string `json:"url"`
	DownloadDetour string `json:"download_detour"`
}

package protocol

import (
	"context"
	"time"

	"github.com/Yat-Muk/prism-v2/internal/pkg/errors"
)

// Type 协议类型
type Type string

const (
	TypeRealityVision Type = "reality_vision"
	TypeRealityGRPC   Type = "reality_grpc"
	TypeHysteria2     Type = "hysteria2"
	TypeTUIC          Type = "tuic"
	TypeAnyTLS        Type = "anytls"
	TypeAnyTLSReality Type = "anytls_reality"
	TypeShadowTLS     Type = "shadowtls"
)

// Protocol 协议接口
type Protocol interface {
	Type() Type
	Name() string
	Port() int
	IsEnabled() bool
	Validate() error

	// 這裡用 map[string]interface{}，和原始 singbox.Inbound 保持一致
	ToSingboxOutbound() (map[string]interface{}, error)
	ToSingboxInbound() (map[string]interface{}, error)
}

// BaseProtocol 协议基础结构
type BaseProtocol struct {
	type_   Type
	name    string
	port    int
	enabled bool
}

func (p *BaseProtocol) Type() Type   { return p.type_ }
func (p *BaseProtocol) Name() string { return p.name }
func (p *BaseProtocol) Port() int    { return p.port }
func (p *BaseProtocol) IsEnabled() bool {
	return p.enabled
}
func (b *BaseProtocol) ValidatePort() error {
	if b.port < 1024 || b.port > 65535 {
		return errors.New("PROTO001", "端口必须在 1024-65535 之间")
	}
	return nil
}

// Service / ServiceStatus / Repository 保持不變
type Service interface {
	Start(ctx context.Context, proto Protocol) error
	Stop(ctx context.Context, proto Protocol) error
	Restart(ctx context.Context, proto Protocol) error
	Status(ctx context.Context, proto Protocol) (*ServiceStatus, error)
}

type ServiceStatus struct {
	Protocol  Type
	Running   bool
	Port      int
	StartTime time.Time
	Error     string
}

type Repository interface {
	Get(ctx context.Context, protoType Type) (Protocol, error)
	Save(ctx context.Context, proto Protocol) error
	List(ctx context.Context) ([]Protocol, error)
	Delete(ctx context.Context, protoType Type) error
}

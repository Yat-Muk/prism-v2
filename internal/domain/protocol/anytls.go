package protocol

import (
	"fmt"

	"github.com/Yat-Muk/prism-v2/internal/pkg/errors"
)

// AnyTLS AnyTLS 协议 (HTTP/HTTPS 伪装)
type AnyTLS struct {
	BaseProtocol
	Username    string // 用户名
	Password    string // 密码
	SNI         string
	CertPath    string   // 证书路径
	KeyPath     string   // 密钥路径
	PaddingMode string   // 填充模式
	ALPN        []string // ALPN
}

// PaddingMode 常量
const (
	PaddingBalanced   = "balanced"
	PaddingMinimal    = "minimal"
	PaddingHighResist = "high_resist"
	PaddingVideo      = "video"
	PaddingOfficial   = "official"
)

// NewAnyTLS 创建 AnyTLS 协议
func NewAnyTLS(port int, password string) *AnyTLS {
	return &AnyTLS{
		BaseProtocol: BaseProtocol{
			type_:   TypeAnyTLS,
			name:    "AnyTLS",
			port:    port,
			enabled: false,
		},
		Username:    "prism",
		Password:    password,
		SNI:         "",
		PaddingMode: PaddingBalanced,
		ALPN:        []string{"h2", "http/1.1"},
	}
}

// Validate 验证配置
func (a *AnyTLS) Validate() error {
	// ✅ 使用统一的端口验证
	if err := a.BaseProtocol.ValidatePort(); err != nil {
		return err
	}

	if a.Password == "" {
		return errors.New("PROTO004", "密码不能为空")
	}

	if a.SNI == "" {
		return errors.New("PROTO009", "SNI 不能为空")
	}

	if a.enabled && a.CertPath == "" {
		return errors.New("PROTO005", "证书路径不能为空")
	}

	if a.enabled && a.KeyPath == "" {
		return errors.New("PROTO006", "密钥路径不能为空")
	}

	return nil
}

// ToSingboxOutbound 转换为 Sing-box outbound 配置
func (a *AnyTLS) ToSingboxOutbound() (map[string]interface{}, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}

	// ✅ 使用构建器
	return NewOutboundBuilder("anytls", "anytls-out", "127.0.0.1", a.port).
		WithAuth("username", a.Username).
		WithAuth("password", a.Password).
		WithTLS(map[string]interface{}{
			"enabled":     true,
			"server_name": a.SNI,
			"alpn":        a.ALPN,
		}).
		Build(), nil
}

// ToSingboxInbound 转换为 Sing-box inbound 配置
func (a *AnyTLS) ToSingboxInbound() (map[string]interface{}, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}

	// ✅ 使用构建器
	return NewInboundBuilder("anytls", "anytls-in", a.port).
		WithUsers([]map[string]interface{}{
			{
				"name":     a.Username,
				"password": a.Password,
			},
		}).
		WithField("padding_scheme", a.getPaddingScheme()).
		WithTLS(map[string]interface{}{
			"enabled":          true,
			"server_name":      a.SNI,
			"certificate_path": a.CertPath,
			"key_path":         a.KeyPath,
			"alpn":             a.ALPN,
		}).
		Build(), nil
}

// getPaddingScheme 获取填充方案
func (a *AnyTLS) getPaddingScheme() []string {
	switch a.PaddingMode {
	case PaddingBalanced:
		return []string{
			"stop=6",
			"0=10-60",
			"1=30-150",
			"2=200-500,c,400-800",
			"3=100-300",
			"4=500-1200",
		}
	case PaddingMinimal:
		return []string{
			"stop=4",
			"0=15-35",
			"1=20-100",
			"2=100-200",
		}
	case PaddingHighResist:
		return []string{
			"stop=10",
			"0=50-100",
			"1=500-800",
			"2=c,800-1200",
			"3=50-50",
			"4=c,1000-1500",
			"5=100-600",
		}
	case PaddingVideo:
		return []string{
			"stop=9",
			"0=40-80",
			"1=600-900",
			"2=c,900-1400",
			"3=80-150",
			"4=c,800-1200",
			"5=200-400",
			"6=150-600",
		}
	case PaddingOfficial:
		fallthrough
	default:
		// 官方默认方案
		return []string{
			"stop=8",
			"0=30-30",
			"1=100-400",
			"2=400-500,c,500-1000,c,500-1000,c,500-1000,c,500-1000",
			"3=9-9,500-1000",
			"4=500-1000",
			"5=500-1000",
			"6=500-1000",
			"7=500-1000",
		}
	}
}

// GenerateShareLink 生成分享链接
func (a *AnyTLS) GenerateShareLink(serverIP string) string {
	return fmt.Sprintf(
		"anytls://%s:%s@%s:%d?security=tls&sni=%s&alpn=%s#AnyTLS",
		a.Username, a.Password, serverIP, a.port, a.SNI, "h2,http/1.1",
	)
}

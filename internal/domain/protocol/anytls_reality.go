package protocol

import (
	"fmt"

	"github.com/Yat-Muk/prism-v2/internal/pkg/errors"
)

// AnyTLSReality AnyTLS Reality 协议
type AnyTLSReality struct {
	BaseProtocol
	Username    string // 用户名
	Password    string // 密码
	SNI         string // TLS SNI
	PublicKey   string // Reality 公钥
	PrivateKey  string // Reality 私钥
	ShortID     string // Short ID 列表
	PaddingMode string // 填充模式
	ALPN        []string
}

// NewAnyTLSReality 创建 AnyTLS Reality 协议
func NewAnyTLSReality(port int, sni, password string) *AnyTLSReality {
	return &AnyTLSReality{
		BaseProtocol: BaseProtocol{
			type_:   TypeAnyTLSReality,
			name:    "AnyTLS Reality",
			port:    port,
			enabled: false,
		},
		Username:    "default",
		Password:    password,
		SNI:         sni,
		PaddingMode: PaddingBalanced,
		ShortID:     "",
	}
}

// Validate 验证配置
func (a *AnyTLSReality) Validate() error {
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

	if a.enabled && a.PrivateKey == "" {
		return errors.New("PROTO012", "Reality 私钥不能为空")
	}
	if a.PublicKey == "" {
		return errors.New("PROTO013", "Reality 公钥不能为空")
	}

	if a.enabled && len(a.ShortID) == 0 {
		return errors.New("PROTO010", "Short ID 不能为空")
	}

	return nil
}

// ToSingboxOutbound 转换为 Sing-box outbound 配置
func (a *AnyTLSReality) ToSingboxOutbound() (map[string]interface{}, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"type":        "anytls",
		"tag":         "anytls-reality-out",
		"server":      "127.0.0.1",
		"server_port": a.port,
		"username":    a.Username,
		"password":    a.Password,
		"tls": map[string]interface{}{
			"enabled":     true,
			"server_name": a.SNI,
			"utls": map[string]interface{}{
				"enabled":     true,
				"fingerprint": "chrome",
			},
			"reality": map[string]interface{}{
				"enabled":    true,
				"public_key": a.PublicKey,
				"short_id":   a.ShortID,
			},
		},
	}, nil
}

// ToSingboxInbound 转换为 Sing-box inbound 配置
func (a *AnyTLSReality) ToSingboxInbound() (map[string]interface{}, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"type":        "anytls",
		"tag":         "anytls-reality-in",
		"listen":      "::",
		"listen_port": a.port,
		"users": []map[string]interface{}{
			{"name": a.Username, "password": a.Password},
		},
		"padding_scheme": a.getPaddingScheme(),
		"tls": map[string]interface{}{
			"enabled":     true,
			"server_name": a.SNI,
			"alpn":        []string{"h2", "http/1.1"},
			"reality": map[string]interface{}{
				"enabled": true,
				"handshake": map[string]interface{}{
					"server":      a.SNI,
					"server_port": 443,
				},
				"private_key": a.PrivateKey,
				"short_id":    []string{a.ShortID},
			},
		},
	}, nil
}

// getPaddingScheme 与 AnyTLS 相同
func (a *AnyTLSReality) getPaddingScheme() []string {
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
func (a *AnyTLSReality) GenerateShareLink(serverIP string) string {
	return fmt.Sprintf(
		"anytls://%s:%s@%s:%d?security=reality&sni=%s&fp=chrome&pbk=%s&sid=%s#AnyTLS-Reality",
		a.Username, a.Password, serverIP, a.port, a.SNI, a.PublicKey, a.ShortID,
	)
}

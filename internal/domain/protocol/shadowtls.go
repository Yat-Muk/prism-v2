package protocol

import (
	"encoding/base64"
	"fmt"

	"github.com/Yat-Muk/prism-v2/internal/pkg/errors"
)

// ShadowTLS ShadowTLS v3 协议
type ShadowTLS struct {
	BaseProtocol
	Password   string // ShadowTLS 密码
	SSPassword string // ShadowSocks 密码
	SSMethod   string // ShadowSocks 加密方法
	SNI        string // TLS SNI
	DetourPort int    // Shadowsocks 监听端口
	StrictMode bool   // 严格模式
}

// NewShadowTLS 创建 ShadowTLS 协议
func NewShadowTLS(port int, password, ssPassword string) *ShadowTLS {
	return &ShadowTLS{
		BaseProtocol: BaseProtocol{
			type_:   TypeShadowTLS,
			name:    "ShadowTLS v3",
			port:    port,
			enabled: false,
		},
		Password:   password,
		SSPassword: ssPassword,
		SSMethod:   "2022-blake3-aes-128-gcm",
		SNI:        "www.microsoft.com",
		DetourPort: 10000,
		StrictMode: true,
	}
}

// Validate 验证配置
func (s *ShadowTLS) Validate() error {
	// ✅ 使用统一的端口验证
	if err := s.BaseProtocol.ValidatePort(); err != nil {
		return err
	}

	if s.Password == "" {
		return errors.New("PROTO004", "ShadowTLS 密码不能为空")
	}

	if s.SSPassword == "" {
		return errors.New("PROTO011", "ShadowSocks 密码不能为空")
	}

	if s.SNI == "" {
		return errors.New("PROTO009", "SNI 不能为空")
	}

	if s.DetourPort < 1 || s.DetourPort > 65535 {
		return errors.New("PROTO001", "Detour 端口范围必须在 1-65535 之间")
	}
	if s.DetourPort == s.port {
		return errors.New("PROTO014", "Detour 端口不能与主端口相同")
	}

	return nil
}

// ToSingboxOutbound 转换为 Sing-box outbound 配置
func (s *ShadowTLS) ToSingboxOutbound() (map[string]interface{}, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}

	// ✅ 使用构建器 - shadowsocks outbound with detour
	return NewOutboundBuilder("shadowsocks", "shadowtls-out", "", 0).
		WithField("detour", "shadowtls-detour").
		WithField("method", s.SSMethod).
		WithField("password", s.SSPassword).
		Build(), nil
}

// GetDetourOutbound 获取 detour outbound 配置
func (s *ShadowTLS) GetDetourOutbound() map[string]interface{} {
	// ✅ 使用构建器
	return NewOutboundBuilder("shadowtls", "shadowtls-detour", "127.0.0.1", s.port).
		WithField("version", 3).
		WithField("password", s.Password).
		WithTLS(map[string]interface{}{
			"enabled":     true,
			"server_name": s.SNI,
		}).
		Build()
}

// ToSingboxInbound 转换为 Sing-box inbound 配置
func (s *ShadowTLS) ToSingboxInbound() (map[string]interface{}, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}

	return NewInboundBuilder("shadowtls", "shadowtls-in", s.Port()).
		WithField("version", 3).
		WithUsers([]map[string]interface{}{
			{"password": s.Password},
		}).
		WithField("handshake", map[string]interface{}{
			"server":      s.SNI,
			"server_port": 443,
		}).
		WithField("detour", "shadowtls-ss-in").
		WithField("strict_mode", s.StrictMode).
		Build(), nil
}

// GetDetourInbound 获取 detour inbound 配置
func (s *ShadowTLS) GetDetourInbound() map[string]interface{} {
	// ✅ 使用构建器（注意：这里 listen 是 127.0.0.1，不是 ::）
	builder := NewInboundBuilder("shadowsocks", "shadowtls-ss-in", s.DetourPort)
	config := builder.Build()
	config["listen"] = "127.0.0.1" // 覆盖默认的 "::"
	config["method"] = s.SSMethod
	config["password"] = s.SSPassword
	return config
}

// GenerateShareLink 生成分享链接
func (s *ShadowTLS) GenerateShareLink(serverIP string) string {
	// ss://method:password@server:port?plugin=shadow-tls;version=3;host=xxx;password=xxx#name
	ssInfo := fmt.Sprintf("%s:%s", s.SSMethod, s.SSPassword)
	ssInfoB64 := base64.StdEncoding.EncodeToString([]byte(ssInfo))

	return fmt.Sprintf(
		"ss://%s@%s:%d?plugin=shadow-tls%%3Bversion%%3D3%%3Bhost%%3D%s%%3Bpassword%%3D%s#ShadowTLS",
		ssInfoB64, serverIP, s.port, s.SNI, s.Password,
	)
}

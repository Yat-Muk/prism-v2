package protocol

import (
	"fmt"

	"github.com/Yat-Muk/prism-v2/internal/pkg/errors"
)

// TUIC TUIC v5 协议
type TUIC struct {
	BaseProtocol
	UUID              string // 用户 UUID
	Password          string // 密码
	SNI               string
	CertPath          string   // 证书路径
	KeyPath           string   // 密钥路径
	ALPN              []string // ALPN
	CongestionControl string   `json:"congestion_control,omitempty"` // bbr
	ZeroRTTHandshake  bool     `json:"zero_rtt_handshake,omitempty"` // false
}

// NewTUIC 创建 TUIC 协议
func NewTUIC(port int, uuid, password string) *TUIC {
	return &TUIC{
		BaseProtocol: BaseProtocol{
			type_:   TypeTUIC,
			name:    "TUIC",
			port:    port,
			enabled: false,
		},
		UUID:              uuid,
		Password:          password,
		SNI:               "",
		ALPN:              []string{"h3"},
		CongestionControl: "bbr",
		ZeroRTTHandshake:  false,
	}
}

// Validate 验证配置
func (t *TUIC) Validate() error {
	// ✅ 使用统一的端口验证
	if err := t.BaseProtocol.ValidatePort(); err != nil {
		return err
	}

	if t.UUID == "" {
		return errors.New("PROTO007", "UUID 不能为空")
	}

	if t.Password == "" {
		return errors.New("PROTO004", "密码不能为空")
	}

	if t.SNI == "" {
		return errors.New("PROTO009", "SNI 不能为空")
	}

	if t.enabled && t.CertPath == "" {
		return errors.New("PROTO005", "证书路径不能为空")
	}

	if t.enabled && t.KeyPath == "" {
		return errors.New("PROTO006", "密钥路径不能为空")
	}

	return nil
}

// ToSingboxOutbound 转换为 Sing-box outbound 配置
func (t *TUIC) ToSingboxOutbound() (map[string]interface{}, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}

	// ✅ 使用构建器
	return NewOutboundBuilder("tuic", "tuic-out", "127.0.0.1", t.port).
		WithAuth("uuid", t.UUID).
		WithAuth("password", t.Password).
		WithTLS(map[string]interface{}{
			"enabled":     true,
			"server_name": t.SNI,
			"alpn":        t.ALPN,
		}).
		WithField("congestion_control", t.CongestionControl).
		WithField("zero_rtt_handshake", t.ZeroRTTHandshake).
		WithField("udp_relay_mode", "native").
		WithField("heartbeat", "10s").
		WithField("network", "tcp,udp").
		Build(), nil
}

// ToSingboxInbound 转换为 Sing-box inbound 配置
func (t *TUIC) ToSingboxInbound() (map[string]interface{}, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}

	// ✅ 使用构建器
	return NewInboundBuilder("tuic", "tuic-in", t.port).
		WithUsers([]map[string]interface{}{
			{
				"name":     "prism",
				"uuid":     t.UUID,
				"password": t.Password,
			},
		}).
		WithTLS(map[string]interface{}{
			"enabled":          true,
			"certificate_path": t.CertPath,
			"key_path":         t.KeyPath,
			"alpn":             t.ALPN,
		}).
		WithField("congestion_control", t.CongestionControl).
		WithField("zero_rtt_handshake", t.ZeroRTTHandshake).
		WithField("auth_timeout", "3s").
		WithField("heartbeat", "10s").
		Build(), nil
}

// GenerateShareLink 生成分享链接
func (t *TUIC) GenerateShareLink(serverIP string) string {
	return fmt.Sprintf(
		"tuic://%s:%s@%s:%d?congestion_control=%s&alpn=%s#TUIC",
		t.UUID, t.Password, serverIP, t.port, t.CongestionControl, t.ALPN[0],
	)
}

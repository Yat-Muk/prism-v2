package protocol

import (
	"fmt"

	"github.com/Yat-Muk/prism-v2/internal/pkg/errors"
)

// RealityGRPC VLESS Reality gRPC 协议
type RealityGRPC struct {
	BaseProtocol
	SNI         string // 伪装域名
	PublicKey   string // 公钥
	PrivateKey  string // 私钥
	ShortID     string // Short ID 列表
	ServiceName string // gRPC 服务名
	Users       []User // 用户列表
}

// NewRealityGRPC 创建 Reality gRPC 协议
func NewRealityGRPC(port int, sni string) *RealityGRPC {
	return &RealityGRPC{
		BaseProtocol: BaseProtocol{
			type_:   TypeRealityGRPC,
			name:    "Reality gRPC",
			port:    port,
			enabled: false,
		},
		SNI:         sni,
		ServiceName: "grpc",
		ShortID:     "",
		Users:       []User{},
	}
}

// Validate 验证配置
func (r *RealityGRPC) Validate() error {
	// 使用统一的端口验证
	if err := r.BaseProtocol.ValidatePort(); err != nil {
		return err
	}

	if r.SNI == "" {
		return errors.New("PROTO002", "SNI 不能为空")
	}

	if r.ServiceName == "" {
		return errors.New("PROTO008", "gRPC 服务名不能为空")
	}

	if len(r.Users) == 0 {
		return errors.New("PROTO003", "至少需要一个用户")
	}

	if r.enabled && r.PrivateKey == "" {
		return errors.New("PROTO012", "Reality 私钥不能为空")
	}
	if r.PublicKey == "" {
		return errors.New("PROTO013", "Reality 公钥不能为空")
	}
	if r.enabled && r.ShortID == "" {
		return errors.New("PROTO010", "Short ID 不能为空")
	}

	return nil
}

// ToSingboxOutbound 转换为 Sing-box outbound 配置
func (r *RealityGRPC) ToSingboxOutbound() (map[string]interface{}, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	// 防禦性編程
	if len(r.Users) == 0 {
		return nil, errors.New("PROTO003", "無法生成 Outbound: 用戶列表為空")
	}

	// 使用构建器
	return NewOutboundBuilder("vless", "reality-grpc-out", "127.0.0.1", r.port).
		WithAuth("uuid", r.Users[0].UUID).
		WithTransport(map[string]interface{}{
			"type":         "grpc",
			"service_name": r.ServiceName,
		}).
		WithTLS(map[string]interface{}{
			"enabled":     true,
			"server_name": r.SNI,
			"utls": map[string]interface{}{
				"enabled":     true,
				"fingerprint": "chrome",
			},
			"reality": map[string]interface{}{
				"enabled":    true,
				"public_key": r.PublicKey,
				"short_id":   r.ShortID,
			},
		}).
		Build(), nil
}

// ToSingboxInbound 转换为 Sing-box inbound 配置
func (r *RealityGRPC) ToSingboxInbound() (map[string]interface{}, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	users := make([]map[string]interface{}, len(r.Users))
	for i, user := range r.Users {
		users[i] = map[string]interface{}{
			"uuid": user.UUID,
		}
	}

	// 使用构建器
	return NewInboundBuilder("vless", "reality-grpc-in", r.port).
		WithUsers(users).
		WithTransport(map[string]interface{}{
			"type":         "grpc",
			"service_name": r.ServiceName,
		}).
		WithTLS(map[string]interface{}{
			"enabled":     true,
			"server_name": r.SNI,
			"reality": map[string]interface{}{
				"enabled": true,
				"handshake": map[string]interface{}{
					"server":      r.SNI,
					"server_port": 443,
				},
				"private_key": r.PrivateKey,
				"short_id":    []string{r.ShortID},
			},
		}).
		Build(), nil
}

// AddUser 添加用户
func (r *RealityGRPC) AddUser(uuid string) {
	r.Users = append(r.Users, User{
		UUID: uuid,
		Flow: "", // gRPC 不需要 flow
	})
}

// GenerateShareLink 生成分享链接
func (r *RealityGRPC) GenerateShareLink(serverIP string, user User) string {
	return fmt.Sprintf(
		"vless://%s@%s:%d?type=grpc&serviceName=%s&security=reality&sni=%s&fp=chrome&pbk=%s&sid=%s#Reality-gRPC",
		user.UUID,
		serverIP,
		r.port,
		r.ServiceName,
		r.SNI,
		r.PublicKey,
		r.ShortID,
	)
}

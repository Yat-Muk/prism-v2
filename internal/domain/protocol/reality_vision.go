package protocol

import (
	"fmt"

	"github.com/Yat-Muk/prism-v2/internal/pkg/errors"
)

// RealityVision Reality Vision 协议
type RealityVision struct {
	BaseProtocol
	SNI        string // 伪装域名
	PublicKey  string // 公钥
	PrivateKey string // 私钥
	ShortID    string // Short ID 列表
	Users      []User // 用户列表
}

// User 用户配置
type User struct {
	UUID string // 用户 UUID
	Flow string // 流控类型
}

// NewRealityVision 创建 Reality Vision 协议
func NewRealityVision(port int, sni string) *RealityVision {
	return &RealityVision{
		BaseProtocol: BaseProtocol{
			type_:   TypeRealityVision,
			name:    "Reality Vision",
			port:    port,
			enabled: false,
		},
		SNI:     sni,
		ShortID: "",
		Users:   []User{},
	}
}

// Validate 验证配置
func (r *RealityVision) Validate() error {
	// 使用统一的端口验证
	if err := r.BaseProtocol.ValidatePort(); err != nil {
		return err
	}

	if r.SNI == "" {
		return errors.New("PROTO002", "SNI 不能为空")
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
func (r *RealityVision) ToSingboxOutbound() (map[string]interface{}, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	// 防禦性編程：再次檢查 Users 長度，防止 Panic
	if len(r.Users) == 0 {
		return nil, errors.New("PROTO003", "無法生成 Outbound: 用戶列表為空")
	}

	// 使用构建器
	return NewOutboundBuilder("vless", "reality-vision-out", "127.0.0.1", r.port).
		WithAuth("uuid", r.Users[0].UUID).
		WithAuth("flow", "xtls-rprx-vision").
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
func (r *RealityVision) ToSingboxInbound() (map[string]interface{}, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	users := make([]map[string]interface{}, len(r.Users))
	for i, user := range r.Users {
		users[i] = map[string]interface{}{
			"uuid": user.UUID,
			"flow": user.Flow,
		}
	}

	// 使用构建器
	return NewInboundBuilder("vless", "reality-vision-in", r.port).
		WithUsers(users).
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
func (r *RealityVision) AddUser(uuid, flow string) {
	r.Users = append(r.Users, User{
		UUID: uuid,
		Flow: flow,
	})
}

// GenerateShareLink 生成分享链接
func (r *RealityVision) GenerateShareLink(serverIP string, user User) string {
	return fmt.Sprintf(
		"vless://%s@%s:%d?type=tcp&security=reality&sni=%s&fp=chrome&pbk=%s&sid=%s&flow=%s#Reality-Vision",
		user.UUID,
		serverIP,
		r.port,
		r.SNI,
		r.PublicKey,
		r.ShortID,
		user.Flow,
	)
}

package protocol

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/pkg/errors"
)

// Hysteria2 Hysteria2 協議
type Hysteria2 struct {
	BaseProtocol
	Password    string // 密碼
	CertPath    string // 證書路徑
	KeyPath     string // 密鑰路徑
	SNI         string
	ALPN        string // ALPN
	Obfs        string // 混淆密碼
	PortHopping string // 端口跳躍範圍 (如: "10000-11000")
	UpMbps      int    // 上行帶寬 (Mbps)
	DownMbps    int    // 下行帶寬 (Mbps)
}

// NewHysteria2 創建 Hysteria2 協議
func NewHysteria2(port int, password string) *Hysteria2 {
	return &Hysteria2{
		BaseProtocol: BaseProtocol{
			type_:   TypeHysteria2,
			name:    "Hysteria2",
			port:    port,
			enabled: false,
		},
		Password: password,
		SNI:      "",
		ALPN:     "h3",
		UpMbps:   100,
		DownMbps: 100,
	}
}

// ParsePortRange 解析端口範圍（支持 - 隔符）
func ParsePortRange(portRange string) (start, end int) {
	parts := strings.Split(portRange, "-")
	if len(parts) != 2 {
		return 0, 0
	}

	s, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	e, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))

	if err1 != nil || err2 != nil {
		return 0, 0
	}

	return s, e
}

// Validate 驗證配置
func (h *Hysteria2) Validate() error {
	// 使用統一的端口驗證
	if err := h.BaseProtocol.ValidatePort(); err != nil {
		return err
	}

	if h.Password == "" {
		return errors.New("PROTO004", "密碼不能為空")
	}

	if h.enabled && h.CertPath == "" {
		return errors.New("PROTO005", "證書路徑不能為空")
	}

	if h.enabled && h.KeyPath == "" {
		return errors.New("PROTO006", "密鑰路徑不能為空")
	}

	if h.UpMbps <= 0 {
		return errors.New("PROTO015", "上行帶寬必須大於 0")
	}
	if h.DownMbps <= 0 {
		return errors.New("PROTO016", "下行帶寬必須大於 0")
	}

	// 驗證端口跳躍格式
	if h.PortHopping != "" {
		start, end := ParsePortRange(h.PortHopping)
		if start == 0 || end == 0 || start >= end {
			return errors.New("PROTO017", fmt.Sprintf("端口跳躍格式錯誤: %s (應為 '20000-30000')", h.PortHopping))
		}
		if start < 1024 || end > 65535 {
			return errors.New("PROTO018", fmt.Sprintf("端口跳躍範圍無效: %s (應在 1024-65535 之間)", h.PortHopping))
		}
	}

	return nil
}

// ToSingboxOutbound 轉換為 Sing-box outbound 配置
func (h *Hysteria2) ToSingboxOutbound() (map[string]interface{}, error) {
	if err := h.Validate(); err != nil {
		return nil, err
	}

	builder := NewOutboundBuilder("hysteria2", "hysteria2-out", "127.0.0.1", h.port).
		WithAuth("password", h.Password).
		WithTLS(map[string]interface{}{
			"enabled":  true,
			"insecure": false,
			"alpn":     []string{h.ALPN},
		})

	if h.Obfs != "" {
		builder.WithField("obfs", map[string]interface{}{
			"type":     "salamander",
			"password": h.Obfs,
		})
	}

	return builder.Build(), nil
}

// ToSingboxInbound 轉換為 Sing-box inbound 配置
func (h *Hysteria2) ToSingboxInbound() (map[string]interface{}, error) {
	if err := h.Validate(); err != nil {
		return nil, err
	}

	builder := NewInboundBuilder("hysteria2", "hysteria2-in", h.port).
		WithUsers([]map[string]interface{}{
			{"password": h.Password},
		}).
		WithTLS(map[string]interface{}{
			"enabled":          true,
			"alpn":             []string{h.ALPN},
			"certificate_path": h.CertPath,
			"key_path":         h.KeyPath,
		})

	if h.Obfs != "" {
		builder.WithField("obfs", map[string]interface{}{
			"type":     "salamander",
			"password": h.Obfs,
		})
	}

	if h.UpMbps > 0 {
		builder.WithField("up_mbps", h.UpMbps)
	}

	if h.DownMbps > 0 {
		builder.WithField("down_mbps", h.DownMbps)
	}

	return builder.Build(), nil
}

// GenerateShareLink 生成分享鏈接
func (h *Hysteria2) GenerateShareLink(serverIP string) string {
	link := fmt.Sprintf("hysteria2://%s@%s:%d", h.Password, serverIP, h.port)

	if h.Obfs != "" {
		link += fmt.Sprintf("?obfs=salamander&obfs-password=%s", h.Obfs)
	}

	link += "#Hysteria2"

	return link
}

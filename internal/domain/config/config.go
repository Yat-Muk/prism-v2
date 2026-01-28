package config

import (
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	mrand "math/rand"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/curve25519"
	"gopkg.in/yaml.v3"

	"github.com/Yat-Muk/prism-v2/internal/pkg/crypto"
)

// Repository 配置倉庫接口
type Repository interface {
	// Load 加載配置
	Load(ctx context.Context) (*Config, error)

	// Save 保存配置
	Save(ctx context.Context, cfg *Config) error
}

// Config 主配置結構
type Config struct {
	Version     int               `yaml:"version" validate:"required,min=2"`
	Server      ServerConfig      `yaml:"server"`
	Log         LogConfig         `yaml:"log"`
	DNS         *DNSConfig        `yaml:"dns,omitempty"`
	UUID        string            `yaml:"uuid"` // 全局用戶標識符（所有協議共用）
	Password    string            `yaml:"password"`
	Protocols   ProtocolsConfig   `yaml:"protocols"` // 所有入站協議相關
	Routing     RoutingConfig     `yaml:"routing"`   // 路由與分流相關
	Backup      BackupConfig      `yaml:"backup"`
	Certificate CertificateConfig `yaml:"certificate"` // 證書配置
}

// ServerConfig 服務器配置
type ServerConfig struct {
	Host string `yaml:"host" validate:"required"`
	Port int    `yaml:"port" validate:"required,min=1,max=65535"`
}

// LogConfig 日誌配置
type LogConfig struct {
	Level      string `yaml:"level" validate:"required,oneof=debug info warn error"`
	OutputPath string `yaml:"output_path"`
	MaxSize    int    `yaml:"max_size" validate:"min=1,max=100"`
	MaxBackups int    `yaml:"max_backups" validate:"min=0,max=30"`
	MaxAge     int    `yaml:"max_age" validate:"min=1,max=365"`
	Compress   bool   `yaml:"compress"`
}

// ProtocolsConfig 協議配置
type ProtocolsConfig struct {
	// 1: VLESS Reality Vision
	RealityVision RealityVisionConfig `yaml:"reality_vision"`
	// 2: VLESS Reality gRPC
	RealityGRPC RealityGRPCConfig `yaml:"reality_grpc"`
	// 3: Hysteria2
	Hysteria2 Hysteria2Config `yaml:"hysteria2"`
	// 4: TUIC v5
	TUIC TUICConfig `yaml:"tuic"`
	// 5: AnyTLS
	AnyTLS AnyTLSConfig `yaml:"anytls"`
	// 6: AnyTLS Reality
	AnyTLSReality AnyTLSRealityConfig `yaml:"anytls_reality"`
	// 7: ShadowTLS v3
	ShadowTLS ShadowTLSConfig `yaml:"shadowtls"`
}

// RealityVisionConfig Reality Vision 配置
type RealityVisionConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Port       int    `yaml:"port" validate:"required_if=Enabled true,omitempty,min=1024,max=65535"`
	SNI        string `yaml:"sni" validate:"required_if=Enabled true,omitempty,fqdn"`
	PublicKey  string `yaml:"public_key"` // 由安裝/更新核心時寫入
	PrivateKey string `yaml:"private_key"`
	ShortID    string `yaml:"short_id"` // 可選，留空則由 sing-box 自行處理
}

// RealityGRPCConfig Reality gRPC 配置
type RealityGRPCConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Port       int    `yaml:"port" validate:"required_if=Enabled true,omitempty,min=1024,max=65535"`
	SNI        string `yaml:"sni" validate:"required_if=Enabled true,omitempty,fqdn"`
	PublicKey  string `yaml:"public_key"`
	PrivateKey string `yaml:"private_key"`
	ShortID    string `yaml:"short_id"`
}

// Hysteria2Config Hysteria2 配置（需要證書）
type Hysteria2Config struct {
	Enabled     bool   `yaml:"enabled"`
	Port        int    `yaml:"port" validate:"required_if=Enabled true,omitempty,min=1024,max=65535"`
	Password    string `yaml:"password" validate:"required_if=Enabled true,omitempty"`
	PortHopping string `yaml:"port_hopping,omitempty"`
	Obfs        string `yaml:"obfs,omitempty"`
	UpMbps      int    `yaml:"up_mbps,omitempty" validate:"omitempty,min=1"`
	DownMbps    int    `yaml:"down_mbps,omitempty" validate:"omitempty,min=1"`
	ALPN        string `yaml:"alpn,omitempty"`
	SNI         string `yaml:"sni,omitempty" validate:"omitempty,fqdn"`

	// 證書模式配置
	CertMode   string `yaml:"cert_mode" validate:"omitempty,oneof=acme self_signed"`
	CertDomain string `yaml:"cert_domain,omitempty"`
}

// TUICConfig TUIC 配置（需要證書）
type TUICConfig struct {
	Enabled           bool     `yaml:"enabled"`
	Port              int      `yaml:"port" validate:"required_if=Enabled true,omitempty,min=1024,max=65535"`
	UUID              string   `yaml:"uuid" validate:"required_if=Enabled true,omitempty,uuid"`
	Password          string   `yaml:"password" validate:"required_if=Enabled true,omitempty"`
	SNI               string   `yaml:"sni,omitempty" validate:"omitempty,fqdn"`
	ALPN              []string `yaml:"alpn,omitempty"`
	CongestionControl string   `yaml:"congestion_control,omitempty"`
	ZeroRTTHandshake  bool     `yaml:"zero_rtt_handshake,omitempty"`

	// 證書模式配置
	CertMode   string `yaml:"cert_mode" validate:"omitempty,oneof=acme self_signed"`
	CertDomain string `yaml:"cert_domain,omitempty"`
}

// AnyTLSConfig AnyTLS 配置（需要證書）
type AnyTLSConfig struct {
	Enabled       bool     `yaml:"enabled"`
	Port          int      `yaml:"port" validate:"required_if=Enabled true,omitempty,min=1024,max=65535"`
	Username      string   `yaml:"username" validate:"required_if=Enabled true,omitempty"`
	Password      string   `yaml:"password" validate:"required_if=Enabled true,omitempty"`
	SNI           string   `yaml:"sni,omitempty" validate:"omitempty,fqdn"`
	PaddingMode   string   `yaml:"padding_mode,omitempty"`
	PaddingScheme []string `yaml:"padding_scheme,omitempty"`
	ALPN          []string `yaml:"alpn,omitempty"`

	// 書模式配置
	CertMode   string `yaml:"cert_mode" validate:"omitempty,oneof=acme self_signed"`
	CertDomain string `yaml:"cert_domain,omitempty"`
}

// AnyTLSRealityConfig AnyTLS Reality 配置
type AnyTLSRealityConfig struct {
	Enabled       bool     `yaml:"enabled"`
	Port          int      `yaml:"port" validate:"required_if=Enabled true,omitempty,min=1024,max=65535"`
	Username      string   `yaml:"username" validate:"required_if=Enabled true,omitempty"`
	Password      string   `yaml:"password" validate:"required_if=Enabled true,omitempty"`
	SNI           string   `yaml:"sni" validate:"required_if=Enabled true,omitempty,fqdn"`
	PublicKey     string   `yaml:"public_key" validate:"required_if=Enabled true,omitempty"`
	PrivateKey    string   `yaml:"private_key" validate:"required_if=Enabled true,omitempty"`
	ShortID       string   `yaml:"short_id,omitempty"`
	PaddingMode   string   `yaml:"padding_mode,omitempty"`
	PaddingScheme []string `yaml:"padding_scheme,omitempty"`
}

// ShadowTLSConfig ShadowTLS v3 配置（✅ 需要證書）
type ShadowTLSConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Port       int    `yaml:"port" validate:"required_if=Enabled true,omitempty,min=1024,max=65535"`
	Password   string `yaml:"password" validate:"required_if=Enabled true,omitempty"`
	SSPassword string `yaml:"ss_password" validate:"required_if=Enabled true,omitempty"`
	SSMethod   string `yaml:"ss_method,omitempty"`
	SNI        string `yaml:"sni,omitempty" validate:"omitempty,fqdn"`
	DetourPort int    `yaml:"detour_port,omitempty" validate:"omitempty,min=1,max=65535"`
	StrictMode bool   `yaml:"strict_mode,omitempty"`
}

// WARPConfig WARP 配置
type WARPConfig struct {
	Enabled    bool     `yaml:"enabled"`
	Global     bool     `yaml:"global"` // true: 全局 WARP，false: 僅匹配 Domains
	Domains    []string `yaml:"domains"`
	PrivateKey string   `yaml:"private_key"`
	IPv6       string   `yaml:"ipv6"`
	Reserved   string   `yaml:"reserved"`
	LicenseKey string   `json:"license_key" yaml:"license_key"`
}

// IPv6SplitConfig IPv6 分流配置
type IPv6SplitConfig struct {
	Enabled bool     `yaml:"enabled"`
	Global  bool     `yaml:"global"`
	Domains []string `yaml:"domains"`
}

// BackupConfig 備份配置
type BackupConfig struct {
	Enabled       bool `yaml:"enabled"`         // 是否啟用自動備份
	MaxFiles      int  `yaml:"max_files"`       // 最大保留文件數
	MaxAgeDays    int  `yaml:"max_age_days"`    // 最大保留天數
	BackupOnApply bool `yaml:"backup_on_apply"` // 保存配置時自動備份
}

// CertificateConfig 證書配置
type CertificateConfig struct {
	// ACME 配置
	ACMEProvider string `yaml:"acme_provider" validate:"omitempty,oneof=letsencrypt zerossl"` // CA 提供商
	ACMEEmail    string `yaml:"acme_email" validate:"omitempty,email"`                        // ACME 郵箱
	ACMEURL      string `yaml:"acme_url"`                                                     // CA URL（可選，默認為生產環境）

	// ZeroSSL EAB (External Account Binding)
	EABKeyID   string `yaml:"eab_key_id"`   // ZeroSSL 需要
	EABHMACKey string `yaml:"eab_hmac_key"` // ZeroSSL 需要

	DNSProvider       string `yaml:"dns_provider" json:"dns_provider"`               // 當前使用的 DNS 提供商
	DNSProviderID     string `yaml:"dns_provider_id" json:"dns_provider_id"`         // DNS API ID
	DNSProviderSecret string `yaml:"dns_provider_secret" json:"dns_provider_secret"` // DNS API Secret

	// DNS Provider 憑證存儲
	DNSProviders map[string]DNSProviderConfig `yaml:"dns_providers"` // key: provider名稱
}

// DNSConfig DNS 配置
type DNSConfig struct {
	Enabled  bool      `yaml:"enabled"`
	Servers  []string  `yaml:"servers"`
	Strategy string    `yaml:"strategy"` // prefer_ipv4, prefer_ipv6, ipv4_only, ipv6_only
	Rules    []DNSRule `yaml:"rules,omitempty"`
}

// DNSRule DNS 規則
type DNSRule struct {
	Domain []string `yaml:"domain,omitempty"`
	Server string   `yaml:"server"`
}

// EncryptSensitiveFields 加密所有敏感字段
func (c *Config) EncryptSensitiveFields(encryptor *crypto.Encryptor) error {
	if encryptor == nil {
		return nil
	}

	// 1. 根密码
	if c.Password != "" && !crypto.IsEncrypted(c.Password) {
		encrypted, err := encryptor.Encrypt(c.Password)
		if err != nil {
			return fmt.Errorf("加密根密码失败: %w", err)
		}
		c.Password = encrypted
	}

	// 2. Reality Vision 私钥
	if c.Protocols.RealityVision.PrivateKey != "" && !crypto.IsEncrypted(c.Protocols.RealityVision.PrivateKey) {
		encrypted, err := encryptor.Encrypt(c.Protocols.RealityVision.PrivateKey)
		if err != nil {
			return fmt.Errorf("加密 RealityVision 私钥失败: %w", err)
		}
		c.Protocols.RealityVision.PrivateKey = encrypted
	}

	// 3. Reality gRPC 私钥
	if c.Protocols.RealityGRPC.PrivateKey != "" && !crypto.IsEncrypted(c.Protocols.RealityGRPC.PrivateKey) {
		encrypted, err := encryptor.Encrypt(c.Protocols.RealityGRPC.PrivateKey)
		if err != nil {
			return fmt.Errorf("加密 RealityGRPC 私钥失败: %w", err)
		}
		c.Protocols.RealityGRPC.PrivateKey = encrypted
	}

	// 4. Hysteria2 密码
	if c.Protocols.Hysteria2.Password != "" && !crypto.IsEncrypted(c.Protocols.Hysteria2.Password) {
		encrypted, err := encryptor.Encrypt(c.Protocols.Hysteria2.Password)
		if err != nil {
			return fmt.Errorf("加密 Hysteria2 密码失败: %w", err)
		}
		c.Protocols.Hysteria2.Password = encrypted
	}

	// 5. TUIC 密码
	if c.Protocols.TUIC.Password != "" && !crypto.IsEncrypted(c.Protocols.TUIC.Password) {
		encrypted, err := encryptor.Encrypt(c.Protocols.TUIC.Password)
		if err != nil {
			return fmt.Errorf("加密 TUIC 密码失败: %w", err)
		}
		c.Protocols.TUIC.Password = encrypted
	}

	// 6. AnyTLS 密码
	if c.Protocols.AnyTLS.Password != "" && !crypto.IsEncrypted(c.Protocols.AnyTLS.Password) {
		encrypted, err := encryptor.Encrypt(c.Protocols.AnyTLS.Password)
		if err != nil {
			return fmt.Errorf("加密 AnyTLS 密码失败: %w", err)
		}
		c.Protocols.AnyTLS.Password = encrypted
	}

	// 7. AnyTLS Reality 私钥和密码
	if c.Protocols.AnyTLSReality.PrivateKey != "" && !crypto.IsEncrypted(c.Protocols.AnyTLSReality.PrivateKey) {
		encrypted, err := encryptor.Encrypt(c.Protocols.AnyTLSReality.PrivateKey)
		if err != nil {
			return fmt.Errorf("加密 AnyTLSReality 私钥失败: %w", err)
		}
		c.Protocols.AnyTLSReality.PrivateKey = encrypted
	}
	if c.Protocols.AnyTLSReality.Password != "" && !crypto.IsEncrypted(c.Protocols.AnyTLSReality.Password) {
		encrypted, err := encryptor.Encrypt(c.Protocols.AnyTLSReality.Password)
		if err != nil {
			return fmt.Errorf("加密 AnyTLSReality 密码失败: %w", err)
		}
		c.Protocols.AnyTLSReality.Password = encrypted
	}

	// 8. ShadowTLS 密码
	if c.Protocols.ShadowTLS.Password != "" && !crypto.IsEncrypted(c.Protocols.ShadowTLS.Password) {
		encrypted, err := encryptor.Encrypt(c.Protocols.ShadowTLS.Password)
		if err != nil {
			return fmt.Errorf("加密 ShadowTLS 密码失败: %w", err)
		}
		c.Protocols.ShadowTLS.Password = encrypted
	}
	if c.Protocols.ShadowTLS.SSPassword != "" && !crypto.IsEncrypted(c.Protocols.ShadowTLS.SSPassword) {
		encrypted, err := encryptor.Encrypt(c.Protocols.ShadowTLS.SSPassword)
		if err != nil {
			return fmt.Errorf("加密 ShadowTLS SS密码失败: %w", err)
		}
		c.Protocols.ShadowTLS.SSPassword = encrypted
	}

	// 9. 证书配置
	if err := c.Certificate.EncryptSensitiveFields(encryptor); err != nil {
		return err
	}

	// 10. WARP
	if err := c.Routing.WARP.EncryptSensitiveFields(encryptor); err != nil {
		return err
	}

	// 11. Socks5
	if err := c.Routing.Socks5.EncryptSensitiveFields(encryptor); err != nil {
		return err
	}

	return nil
}

// DecryptSensitiveFields 解密所有敏感字段
func (c *Config) DecryptSensitiveFields(encryptor *crypto.Encryptor) error {
	if encryptor == nil {
		return nil
	}

	// 1. 根密码
	if crypto.IsEncrypted(c.Password) {
		decrypted, err := encryptor.Decrypt(c.Password)
		if err != nil {
			return fmt.Errorf("解密根密码失败: %w", err)
		}
		c.Password = decrypted
	}

	// 2. Reality Vision 私钥
	if crypto.IsEncrypted(c.Protocols.RealityVision.PrivateKey) {
		decrypted, err := encryptor.Decrypt(c.Protocols.RealityVision.PrivateKey)
		if err != nil {
			return fmt.Errorf("解密 RealityVision 私钥失败: %w", err)
		}
		c.Protocols.RealityVision.PrivateKey = decrypted
	}

	// 3. Reality gRPC 私钥
	if crypto.IsEncrypted(c.Protocols.RealityGRPC.PrivateKey) {
		decrypted, err := encryptor.Decrypt(c.Protocols.RealityGRPC.PrivateKey)
		if err != nil {
			return fmt.Errorf("解密 RealityGRPC 私钥失败: %w", err)
		}
		c.Protocols.RealityGRPC.PrivateKey = decrypted
	}

	// 4. Hysteria2 密码
	if crypto.IsEncrypted(c.Protocols.Hysteria2.Password) {
		decrypted, err := encryptor.Decrypt(c.Protocols.Hysteria2.Password)
		if err != nil {
			return fmt.Errorf("解密 Hysteria2 密码失败: %w", err)
		}
		c.Protocols.Hysteria2.Password = decrypted
	}

	// 5. TUIC 密码
	if crypto.IsEncrypted(c.Protocols.TUIC.Password) {
		decrypted, err := encryptor.Decrypt(c.Protocols.TUIC.Password)
		if err != nil {
			return fmt.Errorf("解密 TUIC 密码失败: %w", err)
		}
		c.Protocols.TUIC.Password = decrypted
	}

	// 6. AnyTLS 密码
	if crypto.IsEncrypted(c.Protocols.AnyTLS.Password) {
		decrypted, err := encryptor.Decrypt(c.Protocols.AnyTLS.Password)
		if err != nil {
			return fmt.Errorf("解密 AnyTLS 密码失败: %w", err)
		}
		c.Protocols.AnyTLS.Password = decrypted
	}

	// 7. AnyTLS Reality 私钥和密码
	if crypto.IsEncrypted(c.Protocols.AnyTLSReality.PrivateKey) {
		decrypted, err := encryptor.Decrypt(c.Protocols.AnyTLSReality.PrivateKey)
		if err != nil {
			return fmt.Errorf("解密 AnyTLSReality 私钥失败: %w", err)
		}
		c.Protocols.AnyTLSReality.PrivateKey = decrypted
	}
	if crypto.IsEncrypted(c.Protocols.AnyTLSReality.Password) {
		decrypted, err := encryptor.Decrypt(c.Protocols.AnyTLSReality.Password)
		if err != nil {
			return fmt.Errorf("解密 AnyTLSReality 密码失败: %w", err)
		}
		c.Protocols.AnyTLSReality.Password = decrypted
	}

	// 8. ShadowTLS 密码
	if crypto.IsEncrypted(c.Protocols.ShadowTLS.Password) {
		decrypted, err := encryptor.Decrypt(c.Protocols.ShadowTLS.Password)
		if err != nil {
			return fmt.Errorf("解密 ShadowTLS 密码失败: %w", err)
		}
		c.Protocols.ShadowTLS.Password = decrypted
	}
	if crypto.IsEncrypted(c.Protocols.ShadowTLS.SSPassword) {
		decrypted, err := encryptor.Decrypt(c.Protocols.ShadowTLS.SSPassword)
		if err != nil {
			return fmt.Errorf("解密 ShadowTLS SS密码失败: %w", err)
		}
		c.Protocols.ShadowTLS.SSPassword = decrypted
	}

	// 9. 证书配置
	if err := c.Certificate.DecryptSensitiveFields(encryptor); err != nil {
		return err
	}

	// 10. WARP
	if err := c.Routing.WARP.DecryptSensitiveFields(encryptor); err != nil {
		return err
	}

	// 11. Socks5
	if err := c.Routing.Socks5.DecryptSensitiveFields(encryptor); err != nil {
		return err
	}

	return nil
}

// EncryptSensitiveFields 加密 WARP 私鑰
func (w *WARPConfig) EncryptSensitiveFields(encryptor *crypto.Encryptor) error {
	if encryptor == nil || w.PrivateKey == "" {
		return nil
	}

	if !crypto.IsEncrypted(w.PrivateKey) {
		encrypted, err := encryptor.Encrypt(w.PrivateKey)
		if err != nil {
			return fmt.Errorf("加密 WARP PrivateKey 失敗: %w", err)
		}
		w.PrivateKey = encrypted
	}

	return nil
}

// DecryptSensitiveFields 解密 WARP 私鑰
func (w *WARPConfig) DecryptSensitiveFields(encryptor *crypto.Encryptor) error {
	if encryptor == nil || w.PrivateKey == "" {
		return nil
	}

	if crypto.IsEncrypted(w.PrivateKey) {
		decrypted, err := encryptor.Decrypt(w.PrivateKey)
		if err != nil {
			return fmt.Errorf("解密 WARP PrivateKey 失敗: %w", err)
		}
		w.PrivateKey = decrypted
	}

	return nil
}

// EncryptSensitiveFields 加密敏感字段
func (c *CertificateConfig) EncryptSensitiveFields(encryptor *crypto.Encryptor) error {
	if encryptor == nil {
		return nil
	}

	// 加密 EAB HMAC Key (ZeroSSL)
	if c.EABHMACKey != "" && !crypto.IsEncrypted(c.EABHMACKey) {
		encrypted, err := encryptor.Encrypt(c.EABHMACKey)
		if err != nil {
			return fmt.Errorf("加密 EAB HMAC Key 失敗: %w", err)
		}
		c.EABHMACKey = encrypted
	}

	// 加密 DNS Provider Secrets
	for name, provider := range c.DNSProviders {
		if provider.Secret != "" && !crypto.IsEncrypted(provider.Secret) {
			encrypted, err := encryptor.Encrypt(provider.Secret)
			if err != nil {
				return fmt.Errorf("加密 DNS Provider (%s) Secret 失敗: %w", name, err)
			}
			provider.Secret = encrypted
			c.DNSProviders[name] = provider
		}
	}

	return nil
}

// DecryptSensitiveFields 解密敏感字段
func (c *CertificateConfig) DecryptSensitiveFields(encryptor *crypto.Encryptor) error {
	if encryptor == nil {
		return nil
	}

	// 解密 EAB HMAC Key
	if c.EABHMACKey != "" && crypto.IsEncrypted(c.EABHMACKey) {
		decrypted, err := encryptor.Decrypt(c.EABHMACKey)
		if err != nil {
			return fmt.Errorf("解密 EAB HMAC Key 失敗: %w", err)
		}
		c.EABHMACKey = decrypted
	}

	// 解密 DNS Provider Secrets
	for name, provider := range c.DNSProviders {
		if provider.Secret != "" && crypto.IsEncrypted(provider.Secret) {
			decrypted, err := encryptor.Decrypt(provider.Secret)
			if err != nil {
				return fmt.Errorf("解密 DNS Provider (%s) Secret 失敗: %w", name, err)
			}
			provider.Secret = decrypted
			c.DNSProviders[name] = provider
		}
	}

	return nil
}

// DNSProviderConfig DNS Provider 配置
type DNSProviderConfig struct {
	ID     string `yaml:"id"`     // API Key / Access ID
	Secret string `yaml:"secret"` // API Secret
}

// DefaultConfig 返回默認配置
func DefaultConfig() *Config {
	// 1. 用 crypto/rand 生成安全密碼（32字節 → Base64）
	cryptoRandomBytes := make([]byte, 32)
	if _, err := crand.Read(cryptoRandomBytes); err != nil {
		panic(err.Error())
	}
	defaultPassword := base64.StdEncoding.EncodeToString(cryptoRandomBytes)

	// 2. 用 math/rand 生成隨機端口
	mrand.Seed(time.Now().UnixNano())
	randomPort := func() int {
		max := big.NewInt(55536)
		n, err := crand.Int(crand.Reader, max)
		if err != nil {
			panic("failed to generate random port: " + err.Error())
		}
		return 10000 + int(n.Int64())
	}

	// 3. 生成 Reality 密鑰對和 ShortID
	realityPrivateKey, realityPublicKey := generateRealityKeypair()
	realityShortID := generateShortID()

	return &Config{
		Version: 2,
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 443,
		},
		Log: LogConfig{
			Level:      "info",
			OutputPath: "/var/log/prism/prism.log",
			MaxSize:    10,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   true,
		},
		UUID:     uuid.New().String(),
		Password: defaultPassword,
		Backup: BackupConfig{
			Enabled:       true,
			MaxFiles:      30,
			MaxAgeDays:    30,
			BackupOnApply: true,
		},

		Protocols: ProtocolsConfig{
			RealityVision: RealityVisionConfig{
				Enabled:    true,
				Port:       randomPort(),
				SNI:        "www.microsoft.com",
				PublicKey:  realityPublicKey,
				PrivateKey: realityPrivateKey,
				ShortID:    realityShortID,
			},
			RealityGRPC: RealityGRPCConfig{
				Enabled:    false,
				Port:       randomPort(),
				SNI:        "www.microsoft.com",
				PublicKey:  realityPublicKey,
				PrivateKey: realityPrivateKey,
				ShortID:    realityShortID,
			},
			Hysteria2: Hysteria2Config{
				Enabled:     true,
				Port:        randomPort(),
				Password:    "",
				PortHopping: "",
				Obfs:        "",
				UpMbps:      100,
				DownMbps:    100,
				ALPN:        "h3",
				SNI:         "www.bing.com",
				CertMode:    "self_signed",
				CertDomain:  "",
			},
			TUIC: TUICConfig{
				Enabled:           true,
				Port:              randomPort(),
				UUID:              "",
				Password:          "",
				SNI:               "www.bing.com",
				ALPN:              []string{"h3"},
				CongestionControl: "bbr",
				ZeroRTTHandshake:  false,
				CertMode:          "self_signed",
				CertDomain:        "",
			},
			AnyTLS: AnyTLSConfig{
				Enabled:       false,
				Port:          randomPort(),
				Username:      "",
				Password:      "",
				SNI:           "www.bing.com",
				PaddingMode:   "official",
				PaddingScheme: nil,
				ALPN:          []string{"h2", "http/1.1"},
				CertMode:      "self_signed",
				CertDomain:    "",
			},
			AnyTLSReality: AnyTLSRealityConfig{
				Enabled:       false,
				Port:          randomPort(),
				Username:      "",
				Password:      "",
				SNI:           "www.microsoft.com",
				PublicKey:     realityPublicKey,
				PrivateKey:    realityPrivateKey,
				ShortID:       "",
				PaddingMode:   "official",
				PaddingScheme: nil,
			},
			ShadowTLS: ShadowTLSConfig{
				Enabled:    false,
				Port:       randomPort(),
				Password:   "",
				SSPassword: "",
				SSMethod:   "2022-blake3-aes-128-gcm",
				SNI:        "www.microsoft.com",
				DetourPort: 10000,
				StrictMode: true,
			},
		},
		Routing: RoutingConfig{
			WARP: WARPConfig{
				Enabled:    false,
				Global:     false,
				Domains:    []string{},
				PrivateKey: "",
				IPv6:       "",
				Reserved:   "",
			},
			IPv6Split: IPv6SplitConfig{
				Enabled: false,
				Global:  false,
				Domains: []string{},
			},
			Socks5: Socks5Config{
				Inbound: Socks5InboundConfig{
					Enabled:        false,
					Port:           1080,
					Username:       "",
					Password:       "",
					AllowedIPs:     []string{},
					AllowAllDomain: false,
					DomainRules:    []string{},
					DomainStrategy: "ipv4_only",
				},
				Outbound: Socks5OutboundConfig{
					Enabled:     false,
					Server:      "",
					Port:        1080,
					Username:    "",
					Password:    "",
					GlobalRoute: false,
					DomainRules: []string{},
				},
			},
			DNS: DNSRoutingConfig{
				Enabled:     false,
				Server:      "",
				DomainRules: []string{},
			},
			SNIProxy: SNIProxyConfig{
				Enabled:     false,
				TargetIP:    "",
				DomainRules: []string{},
			},
			DomainStrategy: "prefer_ipv4",
		},

		Certificate: CertificateConfig{
			ACMEProvider: "letsencrypt",
			ACMEEmail:    "",
			ACMEURL:      "",
			EABKeyID:     "",
			EABHMACKey:   "",
			DNSProviders: make(map[string]DNSProviderConfig),
		},
	}
}

// 生成 Reality 密鑰對（使用 X25519）
func generateRealityKeypair() (privateKey, publicKey string) {
	privateKeyBytes := make([]byte, 32)
	if _, err := crand.Read(privateKeyBytes); err != nil {
		panic("failed to generate reality private key: " + err.Error())
	}

	publicKeyBytes, err := curve25519.X25519(privateKeyBytes, curve25519.Basepoint)
	if err != nil {
		panic("failed to generate reality public key: " + err.Error())
	}

	privateKey = base64.RawURLEncoding.EncodeToString(privateKeyBytes)
	publicKey = base64.RawURLEncoding.EncodeToString(publicKeyBytes)

	return privateKey, publicKey
}

// 生成 Reality ShortID（8 位十六進制）
func generateShortID() string {
	bytes := make([]byte, 8)
	if _, err := crand.Read(bytes); err != nil {
		panic("failed to generate short id: " + err.Error())
	}
	return hex.EncodeToString(bytes)
}

// Validate 驗證配置
func (c *Config) Validate() error {
	return nil
}

// DeepCopy 深拷貝配置 (重構版：序列化回環策略)
// 邏輯：Marshal -> Unmarshal
// 優點：
// 1. 100% 安全：自動處理所有切片、映射和指針，無需人工維護。
// 2. 零維護：新增字段會自動生效。
// 3. 一致性：保證內存副本與磁盤保存的行為完全一致。
// 缺點：性能低於手寫代碼，但由於只在用戶修改配置(Write)時觸發，延遲可忽略不計。
func (c *Config) DeepCopy() *Config {
	if c == nil {
		return nil
	}

	// 1. 序列化為字節
	data, err := yaml.Marshal(c)
	if err != nil {
		// 這種 panic 幾乎不可能發生，除非結構體中有不支持序列化的類型（如 func, chan）
		// 對於 Config 對象來說，這是嚴重編程錯誤，Panic 是正確的
		panic(fmt.Errorf("DeepCopy 序列化失敗 (這是一個 Bug): %w", err))
	}

	// 2. 反序列化為新對象
	var newCfg Config
	if err := yaml.Unmarshal(data, &newCfg); err != nil {
		panic(fmt.Errorf("DeepCopy 反序列化失敗 (這是一個 Bug): %w", err))
	}

	return &newCfg
}

// GetACMEURL 獲取 ACME URL（如果為空則返回默認值）
func (c *CertificateConfig) GetACMEURL() string {
	if c.ACMEURL != "" {
		return c.ACMEURL
	}

	switch c.ACMEProvider {
	case "zerossl":
		return "https://acme.zerossl.com/v2/DV90"
	case "letsencrypt":
		fallthrough
	default:
		return "https://acme-v02.api.letsencrypt.org/directory"
	}
}

// GetDNSProvider 獲取指定的 DNS Provider 配置
func (c *CertificateConfig) GetDNSProvider(name string) (DNSProviderConfig, bool) {
	provider, ok := c.DNSProviders[name]
	return provider, ok
}

// SetDNSProvider 設置 DNS Provider 配置
func (c *CertificateConfig) SetDNSProvider(name, id, secret string) {
	if c.DNSProviders == nil {
		c.DNSProviders = make(map[string]DNSProviderConfig)
	}
	c.DNSProviders[name] = DNSProviderConfig{
		ID:     id,
		Secret: secret,
	}
}

// FillDefaults 自動填充協議默認值
func (c *Config) FillDefaults() {
	// Hysteria2
	if c.Protocols.Hysteria2.Password == "" {
		c.Protocols.Hysteria2.Password = c.Password
	}

	// TUIC
	if c.Protocols.TUIC.UUID == "" {
		c.Protocols.TUIC.UUID = c.UUID
	}
	if c.Protocols.TUIC.Password == "" {
		c.Protocols.TUIC.Password = c.Password
	}

	// AnyTLS
	if c.Protocols.AnyTLS.Username == "" {
		c.Protocols.AnyTLS.Username = "prism"
	}
	if c.Protocols.AnyTLS.Password == "" {
		c.Protocols.AnyTLS.Password = c.Password
	}

	// AnyTLS-Reality
	if c.Protocols.AnyTLSReality.Username == "" {
		c.Protocols.AnyTLSReality.Username = "prism"
	}
	if c.Protocols.AnyTLSReality.Password == "" {
		c.Protocols.AnyTLSReality.Password = c.Password
	}

	// ShadowTLS
	if c.Protocols.ShadowTLS.Password == "" {
		c.Protocols.ShadowTLS.Password = c.Password
	}
	if c.Protocols.ShadowTLS.SSPassword == "" {
		c.Protocols.ShadowTLS.SSPassword = c.Password
	}
}

func (h *Hysteria2Config) ValidatePortHopping() error {
	if h.PortHopping == "" {
		return nil
	}

	var start, end int
	if _, err := fmt.Sscanf(h.PortHopping, "%d-%d", &start, &end); err == nil {
		if start < 1024 || end > 65535 || start >= end {
			return fmt.Errorf("端口跳躍範圍無效: %s", h.PortHopping)
		}
		return nil
	}

	return fmt.Errorf("端口跳躍格式錯誤: %s", h.PortHopping)
}

// ========================================
// 高級路由配置
// ========================================

// RoutingConfig 路由配置
type RoutingConfig struct {
	WARP           WARPConfig       `yaml:"warp"`
	IPv6Split      IPv6SplitConfig  `yaml:"ipv6_split"`
	Socks5         Socks5Config     `yaml:"socks5"`
	DNS            DNSRoutingConfig `yaml:"dns_routing"`
	SNIProxy       SNIProxyConfig   `yaml:"sni_proxy"`
	DomainStrategy string           `yaml:"domain_strategy,omitempty" validate:"omitempty,oneof=prefer_ipv4 prefer_ipv6 ipv4_only ipv6_only"`
}

// Socks5Config Socks5 分流配置
type Socks5Config struct {
	// 入站配置（解鎖機、落地機）
	Inbound Socks5InboundConfig `yaml:"inbound"`
	// 出站配置（轉發機、代理機）
	Outbound Socks5OutboundConfig `yaml:"outbound"`
}

// Socks5InboundConfig Socks5 入站配置
type Socks5InboundConfig struct {
	Enabled        bool     `yaml:"enabled"`
	Port           int      `yaml:"port" validate:"omitempty,min=1024,max=65535"`
	Username       string   `yaml:"username"`
	Password       string   `yaml:"password"`
	AllowedIPs     []string `yaml:"allowed_ips"`      // 允許訪問的 IP 地址
	AllowAllDomain bool     `yaml:"allow_all_domain"` // 是否允許所有域名
	DomainRules    []string `yaml:"domain_rules"`     // 分流域名規則
	DomainStrategy string   `yaml:"domain_strategy" validate:"omitempty,oneof=ipv4_only ipv6_only"`
}

// Socks5OutboundConfig Socks5 出站配置
type Socks5OutboundConfig struct {
	Enabled     bool     `yaml:"enabled"`
	Server      string   `yaml:"server" validate:"omitempty,ip|fqdn"`
	Port        int      `yaml:"port" validate:"omitempty,min=1,max=65535"`
	Username    string   `yaml:"username"`
	Password    string   `yaml:"password"`
	GlobalRoute bool     `yaml:"global_route"` // 全局轉發
	DomainRules []string `yaml:"domain_rules"` // 分流域名規則
}

// DNSRoutingConfig DNS 分流配置
type DNSRoutingConfig struct {
	Enabled     bool     `yaml:"enabled"`
	Server      string   `yaml:"server" validate:"omitempty,ip"` // DNS 服務器 IP
	DomainRules []string `yaml:"domain_rules"`                   // 分流域名規則
}

// SNIProxyConfig SNI 反向代理分流配置
type SNIProxyConfig struct {
	Enabled     bool     `yaml:"enabled"`
	TargetIP    string   `yaml:"target_ip" validate:"omitempty,ip"` // 反向代理目標 IP
	DomainRules []string `yaml:"domain_rules"`                      // 分流域名規則
}

// ========================================
// 加密/解密方法
// ========================================

// EncryptSensitiveFields 加密 Socks5 密碼
func (s *Socks5Config) EncryptSensitiveFields(encryptor *crypto.Encryptor) error {
	if encryptor == nil {
		return nil
	}

	// 加密入站密碼
	if s.Inbound.Password != "" && !crypto.IsEncrypted(s.Inbound.Password) {
		encrypted, err := encryptor.Encrypt(s.Inbound.Password)
		if err != nil {
			return fmt.Errorf("加密 Socks5 入站密碼失敗: %w", err)
		}
		s.Inbound.Password = encrypted
	}

	// 加密出站密碼
	if s.Outbound.Password != "" && !crypto.IsEncrypted(s.Outbound.Password) {
		encrypted, err := encryptor.Encrypt(s.Outbound.Password)
		if err != nil {
			return fmt.Errorf("加密 Socks5 出站密碼失敗: %w", err)
		}
		s.Outbound.Password = encrypted
	}

	return nil
}

// DecryptSensitiveFields 解密 Socks5 密碼
func (s *Socks5Config) DecryptSensitiveFields(encryptor *crypto.Encryptor) error {
	if encryptor == nil {
		return nil
	}

	// 解密入站密碼
	if s.Inbound.Password != "" && crypto.IsEncrypted(s.Inbound.Password) {
		decrypted, err := encryptor.Decrypt(s.Inbound.Password)
		if err != nil {
			return fmt.Errorf("解密 Socks5 入站密碼失敗: %w", err)
		}
		s.Inbound.Password = decrypted
	}

	// 解密出站密碼
	if s.Outbound.Password != "" && crypto.IsEncrypted(s.Outbound.Password) {
		decrypted, err := encryptor.Decrypt(s.Outbound.Password)
		if err != nil {
			return fmt.Errorf("解密 Socks5 出站密碼失敗: %w", err)
		}
		s.Outbound.Password = decrypted
	}

	return nil
}

// ========================================
// 輔助方法
// ========================================

// AddSocks5InboundRule 添加 Socks5 入站分流規則
func (s *Socks5InboundConfig) AddSocks5InboundRule(domain string) {
	// 檢查是否已存在
	for _, existing := range s.DomainRules {
		if existing == domain {
			return
		}
	}
	s.DomainRules = append(s.DomainRules, domain)
}

// RemoveSocks5InboundRule 移除 Socks5 入站分流規則
func (s *Socks5InboundConfig) RemoveSocks5InboundRule(domain string) {
	newRules := make([]string, 0)
	for _, rule := range s.DomainRules {
		if rule != domain {
			newRules = append(newRules, rule)
		}
	}
	s.DomainRules = newRules
}

// AddSocks5OutboundRule 添加 Socks5 出站分流規則
func (s *Socks5OutboundConfig) AddSocks5OutboundRule(domain string) {
	for _, existing := range s.DomainRules {
		if existing == domain {
			return
		}
	}
	s.DomainRules = append(s.DomainRules, domain)
}

// RemoveSocks5OutboundRule 移除 Socks5 出站分流規則
func (s *Socks5OutboundConfig) RemoveSocks5OutboundRule(domain string) {
	newRules := make([]string, 0)
	for _, rule := range s.DomainRules {
		if rule != domain {
			newRules = append(newRules, rule)
		}
	}
	s.DomainRules = newRules
}

// AddDNSRoutingRule 添加 DNS 分流規則
func (d *DNSRoutingConfig) AddDNSRoutingRule(domain string) {
	for _, existing := range d.DomainRules {
		if existing == domain {
			return
		}
	}
	d.DomainRules = append(d.DomainRules, domain)
}

// AddSNIProxyRule 添加 SNI 反向代理規則
func (s *SNIProxyConfig) AddSNIProxyRule(domain string) {
	for _, existing := range s.DomainRules {
		if existing == domain {
			return
		}
	}
	s.DomainRules = append(s.DomainRules, domain)
}

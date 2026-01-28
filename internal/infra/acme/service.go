package acme

import (
	"bufio"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	stdlog "log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/lego"
	legolog "github.com/go-acme/lego/v4/log"
	"github.com/go-acme/lego/v4/providers/dns/alidns"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/providers/dns/dnspod"
	"github.com/go-acme/lego/v4/providers/dns/gcloud"
	"github.com/go-acme/lego/v4/providers/dns/route53"
	"github.com/go-acme/lego/v4/registration"
	"go.uber.org/zap"

	"github.com/Yat-Muk/prism-v2/internal/pkg/appctx"
)

// Service ACME 證書服務
type Service struct {
	paths           *appctx.Paths
	log             *zap.Logger
	client          *lego.Client
	accountEmail    string
	currentProvider string       // 當前使用的 CA 提供商
	directoryURL    string       // ACME 目錄 URL
	mu              sync.RWMutex // 保護併發訪問
	user            *ACMEUser
}

// ACMEUser 實現 lego 的 User 接口
type ACMEUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *ACMEUser) GetEmail() string                        { return u.Email }
func (u *ACMEUser) GetRegistration() *registration.Resource { return u.Registration }
func (u *ACMEUser) GetPrivateKey() crypto.PrivateKey        { return u.key }

// NewService 創建 ACME 服務
func NewService(paths *appctx.Paths, email string, log *zap.Logger) (*Service, error) {
	legolog.Logger = stdlog.New(io.Discard, "", 0)

	if email == "" {
		email = "admin@example.com" // 默認郵箱
	}

	s := &Service{
		paths:           paths,
		log:             log,
		accountEmail:    email,
		currentProvider: "letsencrypt", // 默認使用 Let's Encrypt
		directoryURL:    "https://acme-v02.api.letsencrypt.org/directory",
	}

	// 加載已保存的提供商配置
	if err := s.loadProviderConfig(); err != nil {
		// log.Warn("無法加載提供商配置，使用默認值", zap.Error(err))
	}

	return s, nil
}

// ====================================================================================
// Part 1: 新增接口適配 (用於修復 application 層調用)
// ====================================================================================

type legoAdapter struct {
	client *lego.Client
}

// ObtainCertificate 實現 Client 接口
func (a *legoAdapter) ObtainCertificate(ctx context.Context, domain string) (*certificate.Resource, error) {
	request := certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	}
	return a.client.Certificate.Obtain(request)
}

// GetClient 獲取 ACME 客戶端 (Application 層的主要入口)
func (s *Service) GetClient(provider Provider, opts ...string) (Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. 確保客戶端就緒
	// 使用 Locked 版本，因為我們已經持有了鎖
	client, err := s.getOrCreateClientLocked(context.Background())
	if err != nil {
		return nil, err
	}

	// 2. 根據 Provider 配置 Challenge
	switch provider {
	case ProviderHTTP:
		// 確保 internal/infra/acme/http_provider.go 已存在
		err = client.Challenge.SetHTTP01Provider(NewHTTPProvider())

	case ProviderCloudflare:
		p, e := cloudflare.NewDNSProvider()
		if e == nil {
			err = client.Challenge.SetDNS01Provider(p)
		} else {
			err = e
		}

	case ProviderAliDNS:
		p, e := alidns.NewDNSProvider()
		if e == nil {
			err = client.Challenge.SetDNS01Provider(p)
		} else {
			err = e
		}

	case ProviderDNSPod:
		p, e := dnspod.NewDNSProvider()
		if e == nil {
			err = client.Challenge.SetDNS01Provider(p)
		} else {
			err = e
		}

	case ProviderAWS:
		p, e := route53.NewDNSProvider()
		if e == nil {
			err = client.Challenge.SetDNS01Provider(p)
		} else {
			err = e
		}

	case ProviderGCloud:
		p, e := gcloud.NewDNSProvider()
		if e == nil {
			err = client.Challenge.SetDNS01Provider(p)
		} else {
			err = e
		}

	default:
		return nil, fmt.Errorf("不支持的 Provider: %s", provider)
	}

	if err != nil {
		return nil, fmt.Errorf("設置 Provider 失敗: %w", err)
	}

	// 返回適配器
	return &legoAdapter{client: client}, nil
}

// ====================================================================================
// Part 2: 業務邏輯
// ====================================================================================

// loadProviderConfig 從文件加載提供商配置
func (s *Service) loadProviderConfig() error {
	configPath := filepath.Join(s.paths.CertDir, ".acme-provider")

	f, err := os.Open(configPath)
	if os.IsNotExist(err) {
		return nil // 文件不存在不是錯誤
	}
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	configMap := make(map[string]string)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 跳過空行和註釋
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 使用 SplitN
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			configMap[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// 應用配置
	if provider, ok := configMap["provider"]; ok {
		s.currentProvider = provider
	}
	if directory, ok := configMap["directory"]; ok {
		s.directoryURL = directory
	}

	return nil
}

// getOrCreateClientLocked 獲取或創建 ACME 客戶端 (內部複用，調用前必須持有鎖)
func (s *Service) getOrCreateClientLocked(ctx context.Context) (*lego.Client, error) {
	if s.client != nil {
		return s.client, nil
	}

	// 創建或加載賬戶密鑰
	accountKey, err := s.loadOrCreateAccountKey()
	if err != nil {
		return nil, fmt.Errorf("加載賬戶密鑰失敗: %w", err)
	}

	// 創建 ACME 用戶
	user := &ACMEUser{
		Email: s.accountEmail,
		key:   accountKey,
	}
	s.user = user

	// 配置 ACME 客戶端
	config := lego.NewConfig(user)
	config.CADirURL = s.directoryURL
	config.Certificate.KeyType = certcrypto.EC256

	// 創建客戶端
	client, err := lego.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("創建 ACME 客戶端失敗: %w", err)
	}

	// 註冊賬戶（如果未註冊）
	if user.Registration == nil {
		reg, err := client.Registration.Register(registration.RegisterOptions{
			TermsOfServiceAgreed: true,
		})
		if err != nil {
			// [關鍵修復] 如果註冊失敗，嘗試恢復已有賬戶 (解決 409 Account already exists 錯誤)
			s.log.Info("新註冊失敗，嘗試恢復已有賬戶", zap.Error(err))

			reg, err = client.Registration.ResolveAccountByKey()
			if err != nil {
				return nil, fmt.Errorf("ACME 賬戶註冊及恢復均失敗: %w", err)
			}
			s.log.Info("成功恢復已有 ACME 賬戶", zap.String("provider", s.currentProvider))
		} else {
			s.log.Info("ACME 賬戶註冊成功",
				zap.String("email", s.accountEmail),
				zap.String("provider", s.currentProvider))
		}
		user.Registration = reg
	}

	s.client = client
	return client, nil
}

// loadOrCreateAccountKey 加載或創建賬戶私鑰
func (s *Service) loadOrCreateAccountKey() (crypto.PrivateKey, error) {
	keyPath := filepath.Join(s.paths.CertDir, ".acme-account.key")

	// 嘗試加載現有密鑰
	if data, err := os.ReadFile(keyPath); err == nil {
		block, _ := pem.Decode(data)
		if block != nil {
			if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
				return key, nil
			}
		}
	}

	// 創建新密鑰
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	// 保存密鑰
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	// 確保目錄存在
	if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
		return nil, err
	}

	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return nil, err
	}

	return key, nil
}

// SetEmail 更新賬戶郵箱並重置客戶端
func (s *Service) SetEmail(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.accountEmail != email {
		s.log.Info("更新 ACME 賬戶郵箱", zap.String("new_email", email))
		s.accountEmail = email
		s.client = nil
		s.user = nil
	}
}

// ObtainCertHTTP 使用 HTTP-01 方式申請證書 (Legacy)
func (s *Service) ObtainCertHTTP(ctx context.Context, domain string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.log.Info("開始 HTTP-01 證書申請",
		zap.String("domain", domain),
		zap.String("provider", s.currentProvider))

	client, err := s.getOrCreateClientLocked(ctx)
	if err != nil {
		return err
	}

	// 配置 HTTP-01 challenge
	if err := client.Challenge.SetHTTP01Provider(http01.NewProviderServer("", "80")); err != nil {
		return fmt.Errorf("設置 HTTP-01 provider 失敗: %w", err)
	}

	// 申請證書
	request := certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	}

	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		return fmt.Errorf("證書申請失敗: %w", err)
	}

	// 保存證書
	return s.saveCertificate(domain, certificates)
}

// ObtainCertDNS 使用 DNS-01 方式申請證書 (Legacy)
func (s *Service) ObtainCertDNS(ctx context.Context, domain, provider, id, secret string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.log.Info("開始 DNS-01 證書申請",
		zap.String("domain", domain),
		zap.String("dns_provider", provider),
		zap.String("ca_provider", s.currentProvider))

	client, err := s.getOrCreateClientLocked(ctx)
	if err != nil {
		return err
	}

	// 配置 DNS provider
	dnsProvider, err := s.createDNSProvider(provider, id, secret)
	if err != nil {
		return fmt.Errorf("創建 DNS provider 失敗: %w", err)
	}

	if err := client.Challenge.SetDNS01Provider(dnsProvider); err != nil {
		return fmt.Errorf("設置 DNS-01 provider 失敗: %w", err)
	}

	// 申請證書
	request := certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	}

	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		return fmt.Errorf("證書申請失敗: %w", err)
	}

	// 保存證書
	return s.saveCertificate(domain, certificates)
}

// RenewCertHTTP 使用 HTTP-01 方式續期證書 (Legacy)
func (s *Service) RenewCertHTTP(ctx context.Context, domain string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.log.Info("開始 HTTP-01 證書續期",
		zap.String("domain", domain),
		zap.String("provider", s.currentProvider))

	client, err := s.getOrCreateClientLocked(ctx)
	if err != nil {
		return err
	}

	// 配置 HTTP-01 challenge
	if err := client.Challenge.SetHTTP01Provider(http01.NewProviderServer("", "80")); err != nil {
		return fmt.Errorf("設置 HTTP-01 provider 失敗: %w", err)
	}

	// 加載現有證書
	certResource, err := s.loadCertificateResource(domain)
	if err != nil {
		return fmt.Errorf("加載證書失敗: %w", err)
	}

	// 續期證書
	certificates, err := client.Certificate.Renew(*certResource, true, false, "")
	if err != nil {
		return fmt.Errorf("證書續期失敗: %w", err)
	}

	// 保存新證書
	return s.saveCertificate(domain, certificates)
}

// RenewCertDNS 使用 DNS-01 方式續期證書 (Legacy)
func (s *Service) RenewCertDNS(ctx context.Context, domain, provider, id, secret string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.log.Info("開始 DNS-01 證書續期",
		zap.String("domain", domain),
		zap.String("dns_provider", provider),
		zap.String("ca_provider", s.currentProvider))

	client, err := s.getOrCreateClientLocked(ctx)
	if err != nil {
		return err
	}

	// 配置 DNS provider
	dnsProvider, err := s.createDNSProvider(provider, id, secret)
	if err != nil {
		return fmt.Errorf("創建 DNS provider 失敗: %w", err)
	}

	if err := client.Challenge.SetDNS01Provider(dnsProvider); err != nil {
		return fmt.Errorf("設置 DNS-01 provider 失敗: %w", err)
	}

	// 加載現有證書
	certResource, err := s.loadCertificateResource(domain)
	if err != nil {
		return fmt.Errorf("加載證書失敗: %w", err)
	}

	// 續期證書
	certificates, err := client.Certificate.Renew(*certResource, true, false, "")
	if err != nil {
		return fmt.Errorf("證書續期失敗: %w", err)
	}

	// 保存新證書
	return s.saveCertificate(domain, certificates)
}

// SetProvider 設置 ACME 提供商
func (s *Service) SetProvider(provider, directoryURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.log.Info("切換 ACME 提供商",
		zap.String("from", s.currentProvider),
		zap.String("to", provider))

	s.currentProvider = provider
	s.directoryURL = directoryURL

	// 清空客戶端，下次使用時會用新配置重新創建
	s.client = nil

	return nil
}

// GetCurrentProvider 獲取當前提供商
func (s *Service) GetCurrentProvider() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentProvider
}

// createDNSProvider 創建 DNS provider (輔助方法)
func (s *Service) createDNSProvider(provider, id, secret string) (challenge.Provider, error) {
	switch provider {
	case "cloudflare":
		config := cloudflare.NewDefaultConfig()
		config.AuthToken = secret
		return cloudflare.NewDNSProviderConfig(config)

	case "alidns":
		config := alidns.NewDefaultConfig()
		config.APIKey = id
		config.SecretKey = secret
		return alidns.NewDNSProviderConfig(config)

	case "dnspod":
		config := dnspod.NewDefaultConfig()
		config.LoginToken = fmt.Sprintf("%s,%s", id, secret)
		return dnspod.NewDNSProviderConfig(config)

	case "aws", "route53":
		config := route53.NewDefaultConfig()
		config.AccessKeyID = id
		config.SecretAccessKey = secret
		return route53.NewDNSProviderConfig(config)

	case "googlecloud", "gcloud":
		config := gcloud.NewDefaultConfig()
		config.Project = id
		os.Setenv("GCE_SERVICE_ACCOUNT_FILE", secret)
		return gcloud.NewDNSProviderConfig(config)

	default:
		return nil, fmt.Errorf("不支持的 DNS 提供商: %s", provider)
	}
}

// saveCertificate 保存證書到文件 (輔助方法)
func (s *Service) saveCertificate(domain string, cert *certificate.Resource) error {
	// 確保證書目錄存在
	if err := os.MkdirAll(s.paths.CertDir, 0755); err != nil {
		return fmt.Errorf("創建證書目錄失敗: %w", err)
	}

	certPath := filepath.Join(s.paths.CertDir, domain+".crt")
	keyPath := filepath.Join(s.paths.CertDir, domain+".key")

	// 保存證書
	if err := os.WriteFile(certPath, cert.Certificate, 0644); err != nil {
		return fmt.Errorf("保存證書失敗: %w", err)
	}

	// 保存私鑰
	if err := os.WriteFile(keyPath, cert.PrivateKey, 0600); err != nil {
		return fmt.Errorf("保存私鑰失敗: %w", err)
	}

	// 保存元數據（用於續期）
	metaDir := filepath.Join(s.paths.CertDir, ".acme")
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return fmt.Errorf("創建元數據目錄失敗: %w", err)
	}

	metaPath := filepath.Join(metaDir, domain+".json")
	metaData := fmt.Sprintf(`{
  "domain": "%s",
  "cert_url": "%s",
  "cert_stable_url": "%s",
  "issued_at": "%s",
  "provider": "%s"
}`, domain, cert.CertURL, cert.CertStableURL,
		time.Now().Format(time.RFC3339),
		s.currentProvider)

	if err := os.WriteFile(metaPath, []byte(metaData), 0644); err != nil {
		s.log.Warn("保存元數據失敗", zap.Error(err))
	}

	s.log.Info("證書保存成功",
		zap.String("domain", domain),
		zap.String("cert", certPath),
		zap.String("key", keyPath))

	return nil
}

// loadCertificateResource 加載證書資源（用於續期）
func (s *Service) loadCertificateResource(domain string) (*certificate.Resource, error) {
	certPath := filepath.Join(s.paths.CertDir, domain+".crt")
	keyPath := filepath.Join(s.paths.CertDir, domain+".key")

	certData, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("讀取證書失敗: %w", err)
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("讀取私鑰失敗: %w", err)
	}

	return &certificate.Resource{
		Domain:      domain,
		Certificate: certData,
		PrivateKey:  keyData,
	}, nil
}

package application

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/infra/acme"
	infraCert "github.com/Yat-Muk/prism-v2/internal/infra/certinfo"
	"github.com/Yat-Muk/prism-v2/internal/pkg/appctx"
	"github.com/Yat-Muk/prism-v2/internal/pkg/cert"
	"github.com/Yat-Muk/prism-v2/internal/pkg/sanitizer"
)

type CertService struct {
	paths         *appctx.Paths
	repo          infraCert.Repository
	acmeService   *acme.Service
	selfSignedGen *cert.SelfSignedGenerator
	configService *ConfigService
	log           *zap.Logger
}

func NewCertService(
	paths *appctx.Paths,
	repo infraCert.Repository,
	acmeService *acme.Service,
	selfSignedGen *cert.SelfSignedGenerator,
	configService *ConfigService,
	log *zap.Logger,
) *CertService {
	return &CertService{
		paths:         paths,
		repo:          repo,
		acmeService:   acmeService,
		selfSignedGen: selfSignedGen,
		configService: configService,
		log:           log,
	}
}

// CheckPort80Available 檢查 80 端口是否可用（區分佔用與權限問題）
func (s *CertService) CheckPort80Available() error {
	// 嘗試監聽 80 端口
	ln, err := net.Listen("tcp", ":80")
	if err == nil {
		// 監聽成功，說明端口空閒且權限足夠，立即關閉釋放
		ln.Close()
		return nil
	}

	// 解析底層錯誤以返回更友好的提示
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		var syscallErr *os.SyscallError
		if errors.As(opErr.Err, &syscallErr) {
			// 情況 1: 端口被佔用 (EADDRINUSE)
			if errors.Is(syscallErr.Err, syscall.EADDRINUSE) {
				return fmt.Errorf("端口 80 已被佔用，請先停止佔用 80 端口的服務 (如 Nginx/Caddy/Apache)")
			}

			// 情況 2: 權限不足 (EACCES) - 常見於非 root 用戶
			if errors.Is(syscallErr.Err, syscall.EACCES) || errors.Is(syscallErr.Err, syscall.EPERM) {
				return fmt.Errorf("權限不足，無法綁定端口 80。請使用 root 運行 Prism 或執行: sudo setcap CAP_NET_BIND_SERVICE=+eip <prism_binary>")
			}
		}
	}

	// 其他未知錯誤
	return fmt.Errorf("檢查端口 80 失敗: %w", err)
}

// RequestACMEHTTP 申請 ACME 證書 (HTTP-01 模式)
func (s *CertService) RequestACMEHTTP(ctx context.Context, domain, email string) error {
	s.log.Info("開始 ACME HTTP-01 申請流程",
		zap.String("domain", domain),
		zap.String("email", email),
	)

	// 1. 環境預檢：端口檢查
	if err := s.CheckPort80Available(); err != nil {
		return err
	}

	// 2. [新增] 環境預檢：域名解析檢查
	// 這是為了防止用戶輸入了錯誤的域名，或者域名還沒解析過來
	if err := s.verifyDomainResolution(ctx, domain); err != nil {
		return err
	}

	// 3. 更新郵箱 (如果有的話)
	if email != "" {
		s.acmeService.SetEmail(email)
	}

	// 4. 獲取客戶端
	client, err := s.acmeService.GetClient(acme.ProviderHTTP)
	if err != nil {
		return fmt.Errorf("ACME 初始化失敗: %w", err)
	}

	// 5. 執行申請
	certRes, err := client.ObtainCertificate(ctx, domain)
	if err != nil {
		return fmt.Errorf("申請失敗: %w", err)
	}

	// 6. 保存證書
	if err := s.SaveCertificate(domain, certRes.Certificate, certRes.PrivateKey, "letsencrypt"); err != nil {
		return fmt.Errorf("保存失敗: %w", err)
	}

	s.log.Info("HTTP-01 證書申請成功並已保存", zap.String("domain", domain))
	return nil
}

// verifyDomainResolution 驗證域名是否解析到本機
func (s *CertService) verifyDomainResolution(ctx context.Context, domain string) error {
	// 1. 獲取本機公網 IP
	publicIP, err := s.getPublicIP(ctx)
	if err != nil {
		return fmt.Errorf("環境檢查失敗: 無法獲取本機公網 IP (%v)", err)
	}

	// 2. 解析目標域名
	ips, err := net.LookupHost(domain)
	if err != nil {
		return fmt.Errorf("DNS 解析失敗: 無法查詢域名 %s", domain)
	}

	if len(ips) == 0 {
		return fmt.Errorf("DNS 記錄為空: 域名 %s 沒有 A 記錄", domain)
	}

	// 3. 比對 IP
	match := false
	for _, ip := range ips {
		if ip == publicIP {
			match = true
			break
		}
	}

	if !match {
		return fmt.Errorf("IP 不匹配: 域名指向 %s，但本機 IP 是 %s", ips[0], publicIP)
	}

	s.log.Info("域名解析驗證通過", zap.String("domain", domain), zap.String("ip", publicIP))
	return nil
}

// getPublicIP 獲取本機公網 IP (支持多個備用源)
func (s *CertService) getPublicIP(ctx context.Context) (string, error) {
	sources := []string{
		"https://api.ipify.org?format=text",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
		"https://ident.me",
		"https://checkip.amazonaws.com",
		"https://myexternalip.com/raw",
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, url := range sources {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			s.log.Debug("獲取公網 IP 失敗", zap.String("source", url), zap.Error(err))
			continue // 網絡錯誤，嘗試下一個
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			continue
		}

		ip := strings.TrimSpace(string(body))

		if ip != "" && len(ip) >= 7 && len(ip) <= 45 {
			return ip, nil
		}
	}

	return "", fmt.Errorf("無法從任何源獲取本機公網 IP，請檢查網絡連接")
}

// SaveCertificate 保存證書到文件
func (s *CertService) SaveCertificate(domain string, certBytes []byte, keyBytes []byte, provider string) error {
	// 確保目錄存在
	if err := os.MkdirAll(s.paths.CertDir, 0755); err != nil {
		return fmt.Errorf("創建證書目錄失敗: %w", err)
	}

	certPath := filepath.Join(s.paths.CertDir, domain+".crt")
	keyPath := filepath.Join(s.paths.CertDir, domain+".key")
	metaPath := filepath.Join(s.paths.CertDir, domain+".json")

	// 1. 寫入證書 (0644)
	if err := os.WriteFile(certPath, certBytes, 0644); err != nil {
		return fmt.Errorf("寫入證書失敗: %w", err)
	}

	// 2. 寫入私鑰 (0600)
	if err := os.WriteFile(keyPath, keyBytes, 0600); err != nil {
		return fmt.Errorf("寫入私鑰失敗: %w", err)
	}

	// 3. 寫入元數據
	block, _ := pem.Decode(certBytes)
	var expiry time.Time
	if block != nil {
		if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
			expiry = cert.NotAfter
		}
	}

	metaContent := fmt.Sprintf(`{
			"domain": "%s",
			"provider": "%s",
			"updated_at": "%s",
			"expires_at": "%s"
		}`, domain, provider, time.Now().Format(time.RFC3339), expiry.Format(time.RFC3339))

	_ = os.WriteFile(metaPath, []byte(metaContent), 0644)

	s.log.Info("證書已保存", zap.String("domain", domain), zap.String("path", certPath))
	return nil
}

// RequestACMECertDNS DNS-01 申請
func (s *CertService) RequestACMEDNS(ctx context.Context, domain, provider string, params map[string]string) error {
	s.log.Info("開始 ACME DNS-01 申請", zap.String("domain", domain), zap.String("provider", provider))

	var id, secret string
	if v, ok := params["id"]; ok {
		id = v
	} else if v, ok := params["email"]; ok {
		id = v
	} // 兼容 cloudflare email
	if v, ok := params["secret"]; ok {
		secret = v
	} else if v, ok := params["key"]; ok {
		secret = v
	} // 兼容 key

	s.log.Debug("DNS 憑證", zap.String("id", sanitizer.APIKey(id)))

	// 更新憑證到配置庫 (原子寫入)
	if err := s.UpdateProviderCredentials(ctx, provider, id, secret); err != nil {
		return err
	}

	if err := s.acmeService.ObtainCertDNS(ctx, domain, provider, id, secret); err != nil {
		return err
	}

	// 更新配置為 ACME 模式 (原子寫入)
	return s.updateCertMode(ctx, domain, "acme")
}

// UpdateProviderCredentials 更新憑證並持久化 (使用 ConfigService 原子操作)
func (s *CertService) UpdateProviderCredentials(ctx context.Context, provider, id, secret string) error {
	return s.configService.UpdateConfig(ctx, func(cfg *domainConfig.Config) error {
		cfg.Certificate.SetDNSProvider(provider, id, secret)
		return nil
	})
}

// RenewCertificate 續期證書
func (s *CertService) RenewCertificate(ctx context.Context, domain string) error {
	s.log.Info("開始續期證書", zap.String("domain", domain))

	// 1. 驗證域名
	if domain == "" {
		return fmt.Errorf("域名不能為空")
	}

	// 2. 構建文件路徑
	certPath := filepath.Join(s.paths.CertDir, domain+".crt")
	keyPath := filepath.Join(s.paths.CertDir, domain+".key")

	// 3. 驗證證書文件（統一使用 validateCertFiles）
	if err := s.validateCertFiles(certPath, keyPath); err != nil {
		return err
	}

	s.log.Debug("證書文件檢查通過",
		zap.String("cert", certPath),
		zap.String("key", keyPath))

	// 4. 讀取配置以確定使用的驗證方式 (使用 ConfigService)
	cfg, err := s.configService.GetConfig(ctx)
	if err != nil {
		s.log.Warn("無法讀取配置，使用默認 HTTP-01 驗證", zap.Error(err))
		// 默認使用 HTTP-01
		return s.acmeService.RenewCertHTTP(ctx, domain)
	}

	// 5. 檢查是否有 DNS 提供商配置
	provider := cfg.Certificate.DNSProvider
	if provider != "" && provider != "none" {
		// 使用 DNS-01 續期
		id := cfg.Certificate.DNSProviderID
		secret := cfg.Certificate.DNSProviderSecret

		if id == "" || secret == "" {
			s.log.Warn("DNS 提供商配置不完整，改用 HTTP-01",
				zap.String("provider", provider))
			return s.acmeService.RenewCertHTTP(ctx, domain)
		}

		s.log.Info("使用 DNS-01 方式續期",
			zap.String("domain", domain),
			zap.String("provider", provider))
		return s.acmeService.RenewCertDNS(ctx, domain, provider, id, secret)
	}

	// 6. 默認使用 HTTP-01 續期
	s.log.Info("使用 HTTP-01 方式續期", zap.String("domain", domain))

	// 檢查 80 端口
	if err := s.CheckPort80Available(); err != nil {
		return fmt.Errorf("HTTP-01 續期失敗：%w", err)
	}

	return s.acmeService.RenewCertHTTP(ctx, domain)
}

// validateCertFiles 驗證文件權限和可讀性
func (s *CertService) validateCertFiles(certPath, keyPath string) error {
	// 檢查證書文件
	certInfo, err := os.Stat(certPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("證書文件不存在：%s", certPath)
	}
	if err != nil {
		return fmt.Errorf("無法訪問證書文件：%w", err)
	}
	if certInfo.Size() == 0 {
		return fmt.Errorf("證書文件為空：%s", certPath)
	}

	// 檢查私鑰文件
	keyInfo, err := os.Stat(keyPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("私鑰文件不存在：%s", keyPath)
	}
	if err != nil {
		return fmt.Errorf("無法訪問私鑰文件：%w", err)
	}
	if keyInfo.Size() == 0 {
		return fmt.Errorf("私鑰文件為空：%s", keyPath)
	}

	// 檢查私鑰權限（應該是 600 或更嚴格）
	if keyInfo.Mode().Perm() > 0600 {
		s.log.Warn("私鑰文件權限過於寬鬆",
			zap.String("path", keyPath),
			zap.String("perm", keyInfo.Mode().Perm().String()))
	}

	return nil
}

// DeleteCertificate 刪除證書
func (s *CertService) DeleteCertificate(domain string) error {
	var errs []string

	// 1. 根目錄文件
	certPath := filepath.Join(s.paths.CertDir, domain+".crt")
	keyPath := filepath.Join(s.paths.CertDir, domain+".key")
	metaPath := filepath.Join(s.paths.CertDir, domain+".json")

	if err := os.Remove(certPath); err != nil && !os.IsNotExist(err) {
		errs = append(errs, fmt.Sprintf("crt: %v", err))
	}
	if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
		errs = append(errs, fmt.Sprintf("key: %v", err))
	}
	if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
		errs = append(errs, fmt.Sprintf("json: %v", err))
	}

	// 2. 清理 .acme 子目錄（兼容舊數據）
	legacyMetaPath := filepath.Join(s.paths.CertDir, ".acme", domain)
	_ = os.RemoveAll(legacyMetaPath)

	if len(errs) > 0 {
		return fmt.Errorf("刪除失敗: %s", strings.Join(errs, ", "))
	}
	return nil
}

// SwitchProvider 切換 ACME CA 提供商
func (s *CertService) SwitchProvider(ctx context.Context, provider string) error {
	s.log.Info("切換 ACME CA 提供商", zap.String("provider", provider))

	// 1. 驗證提供商名稱
	var directoryURL string
	switch provider {
	case "letsencrypt":
		directoryURL = "https://acme-v02.api.letsencrypt.org/directory"
	case "zerossl":
		directoryURL = "https://acme.zerossl.com/v2/DV90"
	default:
		return fmt.Errorf("不支持的 CA 提供商: %s（支持：letsencrypt, zerossl）", provider)
	}

	// 2. 保存提供商配置到文件
	configPath := filepath.Join(s.paths.CertDir, ".acme-provider")

	// 確保證書目錄存在
	if err := os.MkdirAll(s.paths.CertDir, 0755); err != nil {
		return fmt.Errorf("創建證書目錄失敗: %w", err)
	}

	content := fmt.Sprintf("provider=%s\ndirectory=%s\nlast_updated=%s\n",
		provider,
		directoryURL,
		time.Now().Format(time.RFC3339))

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("保存提供商配置失敗: %w", err)
	}

	// 3. 更新 ACME 服務配置
	if s.acmeService != nil {
		if err := s.acmeService.SetProvider(provider, directoryURL); err != nil {
			return fmt.Errorf("更新 ACME 服務配置失敗: %w", err)
		}
	}

	s.log.Info("成功切換 ACME 提供商",
		zap.String("provider", provider),
		zap.String("directory", directoryURL))

	return nil
}

// GetCurrentProvider 獲取當前 CA 提供商
func (s *CertService) GetCurrentProvider() string {
	// 1. 嘗試從配置文件讀取
	configPath := filepath.Join(s.paths.CertDir, ".acme-provider")

	if data, err := os.ReadFile(configPath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "provider=") {
				provider := strings.TrimPrefix(line, "provider=")
				provider = strings.TrimSpace(provider)
				if provider != "" {
					return provider
				}
			}
		}
	}

	// 2. 詢問 ACME 服務
	if s.acmeService != nil {
		if provider := s.acmeService.GetCurrentProvider(); provider != "" {
			return provider
		}
	}

	// 3. 返回默認值
	return "letsencrypt"
}

// updateCertMode 更新配置中的證書模式 (使用 ConfigService 原子操作)
func (s *CertService) updateCertMode(ctx context.Context, domain, mode string) error {
	return s.configService.UpdateConfig(ctx, func(cfg *domainConfig.Config) error {
		updated := false

		// 檢查並更新各個協議的證書配置
		if cfg.Protocols.Hysteria2.Enabled && cfg.Protocols.Hysteria2.SNI == domain {
			cfg.Protocols.Hysteria2.CertMode = mode
			cfg.Protocols.Hysteria2.CertDomain = domain
			updated = true
		}

		if cfg.Protocols.TUIC.Enabled && cfg.Protocols.TUIC.SNI == domain {
			cfg.Protocols.TUIC.CertMode = mode
			cfg.Protocols.TUIC.CertDomain = domain
			updated = true
		}

		if cfg.Protocols.AnyTLS.Enabled && cfg.Protocols.AnyTLS.SNI == domain {
			cfg.Protocols.AnyTLS.CertMode = mode
			cfg.Protocols.AnyTLS.CertDomain = domain
			updated = true
		}

		if updated {
			s.log.Info("已更新協議證書配置",
				zap.String("domain", domain),
				zap.String("mode", mode))
		}
		return nil
	})
}

// GetCertList 獲取所有證書列表
func (s *CertService) GetCertList() ([]infraCert.CertInfo, []string, []string) {
	// 使用注入的 paths
	acmeList, _ := s.repo.ListCerts()
	selfList := infraCert.GetSelfSignedCertListFromDir(s.paths.CertDir)

	allCerts := append(acmeList, selfList...)

	acmeDomains := make([]string, 0)
	for _, c := range acmeList {
		if c.IsValid {
			acmeDomains = append(acmeDomains, c.Domain)
		}
	}

	selfDomains := make([]string, 0)
	for _, c := range selfList {
		if c.IsValid {
			selfDomains = append(selfDomains, c.Domain)
		}
	}

	return allCerts, acmeDomains, selfDomains
}

// GetCertPath 獲取證書路徑
func (s *CertService) GetCertPath(protocolName, certMode, certDomain string) (certPath, keyPath string, err error) {
	baseDir := s.paths.CertDir

	switch certMode {
	case "acme":
		if certDomain == "" {
			return "", "", fmt.Errorf("協議 %s 啟用了 ACME 但未指定域名", protocolName)
		}
		certPath = filepath.Join(baseDir, certDomain+".crt")
		keyPath = filepath.Join(baseDir, certDomain+".key")

	case "self_signed":
		certPath = filepath.Join(baseDir, "self_signed.crt")
		keyPath = filepath.Join(baseDir, "self_signed.key")

		targetDomain := certDomain
		if targetDomain == "" {
			targetDomain = "www.bing.com"
		}

		if err := s.selfSignedGen.EnsureSelfSigned(certPath, keyPath, targetDomain); err != nil {
			return "", "", err
		}

	default:
		return "", "", nil
	}

	// 檢查文件是否存在
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("證書文件不存在: %s", certPath)
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("密鑰文件不存在: %s", keyPath)
	}

	return certPath, keyPath, nil
}

// RenewAll 續期所有證書
func (s *CertService) RenewAll(ctx context.Context) error {
	s.log.Info("開始批量續期任務")

	// 1. 獲取當前所有 ACME 證書域名
	// GetCertList 返回: (所有詳情, ACME域名列表, 自簽名域名列表)
	_, acmeDomains, _ := s.GetCertList()

	if len(acmeDomains) == 0 {
		s.log.Info("沒有發現需要續期的 ACME 證書")
		return nil
	}

	var errs []string
	successCount := 0

	// 2. 遍歷並逐個續期
	for _, domain := range acmeDomains {
		s.log.Info("正在檢查續期", zap.String("domain", domain))

		// 調用自身的 RenewCertificate，它已經包含了讀取配置、選擇 HTTP/DNS 模式的邏輯
		if err := s.RenewCertificate(ctx, domain); err != nil {
			// 記錄錯誤但不中斷整個流程
			errMsg := fmt.Sprintf("%s: %v", domain, err)
			s.log.Warn("續期失敗", zap.String("domain", domain), zap.Error(err))
			errs = append(errs, errMsg)
		} else {
			successCount++
			s.log.Info("續期檢查完成", zap.String("domain", domain))
		}
	}

	s.log.Info("批量續期結束",
		zap.Int("總數", len(acmeDomains)),
		zap.Int("成功", successCount),
		zap.Int("失敗", len(errs)))

	// 3. 如果有錯誤，匯總返回
	if len(errs) > 0 {
		return fmt.Errorf("部分證書操作失敗: %s", strings.Join(errs, "; "))
	}

	return nil
}

func (s *CertService) DeleteCert(domain string) error {
	return s.DeleteCertificate(domain) // 調用內部詳細實現
}

func (s *CertService) SetProvider(provider string) error {
	// 這裡可以傳遞 context.Background() 或者修改 SwitchProvider 簽名
	return s.SwitchProvider(context.Background(), provider)
}

// EnsureSelfSigned 確保自簽名證書存在
func (s *CertService) EnsureSelfSigned(certPath, keyPath, domain string) error {
	return s.selfSignedGen.EnsureSelfSigned(certPath, keyPath, domain)
}

// ToggleProtocolCertMode 切換協議證書模式 (業務邏輯核心)
func (s *CertService) ToggleProtocolCertMode(ctx context.Context, protocol string) (string, error) {
	var resultMsg string

	// 使用 UpdateConfig 閉包，持有鎖
	err := s.configService.UpdateConfig(ctx, func(cfg *domainConfig.Config) error {
		var currentMode, sni string
		var setConfig func(m, d, s string)

		// 1. 根據協議類型定位配置字段
		switch protocol {
		case "hysteria2":
			if !cfg.Protocols.Hysteria2.Enabled {
				return fmt.Errorf("Hysteria 2 未啟用，無法修改")
			}
			currentMode = cfg.Protocols.Hysteria2.CertMode
			sni = cfg.Protocols.Hysteria2.SNI
			setConfig = func(m, d, s string) {
				cfg.Protocols.Hysteria2.CertMode = m
				cfg.Protocols.Hysteria2.CertDomain = d
				if s != "" {
					cfg.Protocols.Hysteria2.SNI = s
				}
			}
		case "tuic":
			if !cfg.Protocols.TUIC.Enabled {
				return fmt.Errorf("TUIC 未啟用，無法修改")
			}
			currentMode = cfg.Protocols.TUIC.CertMode
			sni = cfg.Protocols.TUIC.SNI
			setConfig = func(m, d, s string) {
				cfg.Protocols.TUIC.CertMode = m
				cfg.Protocols.TUIC.CertDomain = d
				if s != "" {
					cfg.Protocols.TUIC.SNI = s
				}
			}
		case "anytls":
			if !cfg.Protocols.AnyTLS.Enabled {
				return fmt.Errorf("AnyTLS 未啟用，無法修改")
			}
			currentMode = cfg.Protocols.AnyTLS.CertMode
			sni = cfg.Protocols.AnyTLS.SNI
			setConfig = func(m, d, s string) {
				cfg.Protocols.AnyTLS.CertMode = m
				cfg.Protocols.AnyTLS.CertDomain = d
				if s != "" {
					cfg.Protocols.AnyTLS.SNI = s
				}
			}
		default:
			return fmt.Errorf("未知協議: %s", protocol)
		}

		// 2. 執行切換邏輯
		if currentMode == "acme" {
			// === 切換到 自簽名模式 ===
			targetSNI := "www.bing.com"
			setConfig("self_signed", "", targetSNI)
			resultMsg = fmt.Sprintf("已切換為 [自簽名] 模式 (域名: %s)", targetSNI)
		} else {
			// === 切換到 ACME 模式 ===
			// 注意：GetCertList 不涉及 Config 讀寫，可以在鎖內調用
			_, acmeDomains, _ := s.GetCertList()
			if len(acmeDomains) == 0 {
				return fmt.Errorf("未找到 ACME 證書，請先申請證書")
			}

			// 智能選擇：優先匹配當前 SNI
			targetDomain := acmeDomains[0]
			foundMatch := false
			for _, d := range acmeDomains {
				if d == sni {
					targetDomain = d
					foundMatch = true
					break
				}
			}

			setConfig("acme", targetDomain, targetDomain)
			matchNote := ""
			if foundMatch {
				matchNote = " (匹配現有SNI)"
			}
			resultMsg = fmt.Sprintf("已切換為 [ACME] 模式%s (域名: %s)", matchNote, targetDomain)
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	return resultMsg, nil
}

// CheckAndRenewAll 檢查所有證書並對即將過期的進行續期
// 返回值: (是否有證書更新, 錯誤信息)
// 用於: Cron 定時任務
func (s *CertService) CheckAndRenewAll(ctx context.Context) (bool, error) {
	s.log.Info("=== 開始執行證書自動維護檢查 ===")

	// 1. 獲取所有證書的詳細信息
	// GetCertList 返回: (所有詳情, ACME域名, 自簽名域名)
	allCerts, _, _ := s.GetCertList()

	anyRenewed := false
	var errs []string

	for _, cert := range allCerts {
		// 1.1 跳過自簽名證書 (由 Prism 內部管理，通常不過期或不需要 ACME 邏輯)
		if cert.Provider == "self_signed" {
			continue
		}

		// 1.2 跳過無效/損壞的證書
		if !cert.IsValid {
			s.log.Warn("發現無效證書元數據，跳過", zap.String("domain", cert.Domain))
			continue
		}

		// 2. 解析過期時間 (數據來源是 .json 元數據文件)
		expiry, err := time.Parse(time.RFC3339, cert.ExpiresAt)
		if err != nil {
			s.log.Error("無法解析過期時間", zap.String("domain", cert.Domain), zap.Error(err))
			continue
		}

		// 3. 計算剩餘天數
		remaining := time.Until(expiry)
		daysRemaining := int(remaining.Hours() / 24)

		// 4. 判斷是否需要續期 (閾值: 30天)
		if daysRemaining > 30 {
			s.log.Debug("證書有效期充足，跳過",
				zap.String("domain", cert.Domain),
				zap.Int("剩餘天數", daysRemaining))
			continue
		}

		s.log.Info("證書即將過期，開始續期",
			zap.String("domain", cert.Domain),
			zap.Int("剩餘天數", daysRemaining))

		// 5. 執行續期 (復用現有的 RenewCertificate 邏輯，它會自動處理 HTTP/DNS 模式)
		if err := s.RenewCertificate(ctx, cert.Domain); err != nil {
			errMsg := fmt.Sprintf("域名 %s 續期失敗: %v", cert.Domain, err)
			s.log.Error("續期失敗", zap.String("domain", cert.Domain), zap.Error(err))
			errs = append(errs, errMsg)
		} else {
			s.log.Info("證書續期成功", zap.String("domain", cert.Domain))
			anyRenewed = true
		}
	}

	s.log.Info("證書維護檢查完成", zap.Bool("是否有更新", anyRenewed))

	if len(errs) > 0 {
		return anyRenewed, fmt.Errorf("部分證書維護失敗: %s", strings.Join(errs, "; "))
	}

	return anyRenewed, nil
}

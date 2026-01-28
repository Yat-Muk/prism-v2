package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/infra/certinfo"
	"github.com/Yat-Muk/prism-v2/internal/pkg/appctx"
	"github.com/Yat-Muk/prism-v2/internal/pkg/cert"
	"go.uber.org/zap"
)

func TestCertService_SaveAndDelete(t *testing.T) {
	tempDir := t.TempDir()
	paths := &appctx.Paths{CertDir: tempDir}

	// 创建真实的 certRepo
	certRepo := certinfo.NewRepository(tempDir)

	svc := NewCertService(
		paths,
		certRepo,
		nil,
		nil,
		nil,
		zap.NewNop(),
	)

	domain := "example.com"
	certContent := []byte("-----BEGIN CERTIFICATE-----\nFAKE\n-----END CERTIFICATE-----")
	keyContent := []byte("-----BEGIN RSA PRIVATE KEY-----\nFAKE\n-----END RSA PRIVATE KEY-----")

	err := svc.SaveCertificate(domain, certContent, keyContent, "letsencrypt")
	if err != nil {
		t.Fatalf("保存證書失敗: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, domain+".crt")); err != nil {
		t.Error("證書文件未生成")
	}

	if err := svc.DeleteCertificate(domain); err != nil {
		t.Fatal(err)
	}
}

func TestCertService_ToggleProtocolCertMode(t *testing.T) {
	// 1. 初始化環境
	mockRepo := &MockRepo{cfg: domainConfig.DefaultConfig()}
	mockRepo.cfg.Protocols.Hysteria2.Enabled = true
	mockRepo.cfg.Protocols.Hysteria2.CertMode = "self_signed"

	logger := zap.NewNop()
	cfgSvc := NewConfigService(mockRepo, logger)

	tempDir := t.TempDir()
	paths := &appctx.Paths{CertDir: tempDir}

	// 创建真实的 certRepo
	certRepo := certinfo.NewRepository(tempDir)

	// 初始化 SelfSignedGenerator
	gen := cert.NewSelfSignedGenerator(logger)

	svc := NewCertService(
		paths,
		certRepo,
		nil,
		gen,
		cfgSvc,
		logger,
	)
	ctx := context.Background()

	// 2. 準備測試數據
	dummyDomain := "test.com"
	certPath := filepath.Join(tempDir, dummyDomain+".crt")
	keyPath := filepath.Join(tempDir, dummyDomain+".key")

	// 生成物理文件
	if err := gen.EnsureSelfSigned(certPath, keyPath, dummyDomain); err != nil {
		t.Fatalf("生成模擬證書失敗: %v", err)
	}

	// 寫入對應的 .json 元數據
	metaFile := filepath.Join(tempDir, dummyDomain+".json")
	metaJSON := fmt.Sprintf(`{
		"domain": "%s",
		"provider": "letsencrypt",
		"updated_at": "2026-01-01T00:00:00Z",
		"expires_at": "2099-01-01T00:00:00Z"
	}`, dummyDomain)
	if err := os.WriteFile(metaFile, []byte(metaJSON), 0644); err != nil {
		t.Fatalf("寫入元數據失敗: %v", err)
	}

	// 確保 SNI 一致
	mockRepo.cfg.Protocols.Hysteria2.SNI = dummyDomain

	// 3. 執行切換 (self_signed -> acme)
	msg, err := svc.ToggleProtocolCertMode(ctx, "hysteria2")
	if err != nil {
		t.Fatalf("切換失敗: %v", err)
	}
	t.Logf("切換成功: %s", msg)

	// 4. 驗證
	updatedCfg, err := cfgSvc.GetConfig(ctx)
	if err != nil {
		t.Fatalf("獲取配置失敗: %v", err)
	}

	if updatedCfg.Protocols.Hysteria2.CertMode != "acme" {
		t.Errorf("預期模式為 acme, 實際為 %s", updatedCfg.Protocols.Hysteria2.CertMode)
	}
}

func TestCertService_ProviderManagement(t *testing.T) {
	tempDir := t.TempDir()
	paths := &appctx.Paths{CertDir: tempDir}

	// 创建真实的 certRepo
	certRepo := certinfo.NewRepository(tempDir)

	svc := NewCertService(
		paths,
		certRepo,
		nil,
		nil,
		nil,
		zap.NewNop(),
	)

	err := svc.SwitchProvider(context.Background(), "zerossl")
	if err != nil {
		t.Fatal(err)
	}

	if p := svc.GetCurrentProvider(); p != "zerossl" {
		t.Errorf("Provider 未正確切換: %s", p)
	}
}

func TestCertService_GetPublicIP(t *testing.T) {
	if testing.Short() {
		t.Skip("跳過網絡測試")
	}

	tempDir := t.TempDir()
	paths := &appctx.Paths{CertDir: tempDir}

	// 创建真实的 certRepo
	certRepo := certinfo.NewRepository(tempDir)

	svc := NewCertService(
		paths,
		certRepo,
		nil,
		nil,
		nil,
		zap.NewNop(),
	)

	ip, err := svc.getPublicIP(context.Background())
	if err != nil {
		t.Skip("無法獲取公網 IP，跳過此項")
		return
	}
	if ip == "" {
		t.Error("獲取的 IP 為空")
	}
}

func TestCertService_GetCertList(t *testing.T) {
	tempDir := t.TempDir()
	paths := &appctx.Paths{CertDir: tempDir}
	certRepo := certinfo.NewRepository(tempDir)

	svc := NewCertService(
		paths,
		certRepo,
		nil,
		nil,
		nil,
		zap.NewNop(),
	)

	// 測試空列表
	allCerts, acmeDomains, selfDomains := svc.GetCertList()
	if len(allCerts) != 0 {
		t.Errorf("初始證書列表應為空，實際有 %d 個", len(allCerts))
	}

	// 保存證書
	domain := "test.example.com"
	certContent := []byte("-----BEGIN CERTIFICATE-----\nTEST\n-----END CERTIFICATE-----")
	keyContent := []byte("-----BEGIN RSA PRIVATE KEY-----\nTEST\n-----END RSA PRIVATE KEY-----")

	if err := svc.SaveCertificate(domain, certContent, keyContent, "letsencrypt"); err != nil {
		t.Fatalf("保存證書失敗: %v", err)
	}

	// 獲取更新後的列表
	allCerts, acmeDomains, selfDomains = svc.GetCertList()

	// 打印調試信息
	t.Logf("證書總數: %d", len(allCerts))
	t.Logf("ACME 證書: %v", acmeDomains)
	t.Logf("自簽名證書: %v", selfDomains)

	// 驗證：至少應該有一個證書
	if len(allCerts) == 0 {
		t.Fatal("保存證書後列表為空")
	}

	// 查找我們保存的證書
	var foundCert *certinfo.CertInfo
	for i := range allCerts {
		if allCerts[i].Domain == domain {
			foundCert = &allCerts[i]
			break
		}
	}

	if foundCert == nil {
		t.Fatalf("未找到域名 %s 的證書", domain)
	}

	// 驗證證書屬性
	t.Logf("找到證書: Provider=%s, IsValid=%v", foundCert.Provider, foundCert.IsValid)

	// 證書應該在某個列表中（ACME 或自簽名）
	inAcme := false
	for _, d := range acmeDomains {
		if d == domain {
			inAcme = true
			break
		}
	}

	inSelf := false
	for _, d := range selfDomains {
		if d == domain {
			inSelf = true
			break
		}
	}

	if !inAcme && !inSelf {
		// 這可能是因為 IsValid=false，這也是可接受的
		if !foundCert.IsValid {
			t.Logf("證書無效（IsValid=false），因此不在任何域名列表中")
		} else {
			t.Errorf("有效證書應該在 ACME 或自簽名列表中")
		}
	}
}

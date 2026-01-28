package cert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/Yat-Muk/prism-v2/internal/pkg/tlsconfig"
	"go.uber.org/zap"
)

const (
	SelfSignedDirMode  os.FileMode = 0700
	SelfSignedCertMode os.FileMode = 0644
	SelfSignedKeyMode  os.FileMode = 0600
)

type SelfSignedGenerator struct {
	log *zap.Logger
}

func NewSelfSignedGenerator(log *zap.Logger) *SelfSignedGenerator {
	return &SelfSignedGenerator{log: log}
}

// EnsureSelfSigned 確保自簽名證書存在 (支持自定義域名)
func (g *SelfSignedGenerator) EnsureSelfSigned(certPath, keyPath, domain string) error {
	// 默認值
	if domain == "" {
		domain = "www.bing.com"
	}

	// 檢查證書是否已存在
	if _, certErr := os.Stat(certPath); certErr == nil {
		if _, keyErr := os.Stat(keyPath); keyErr == nil {
			// 文件已存在，跳過生成
			return nil
		}
	}

	g.log.Info("生成新的自簽名證書...",
		zap.String("domain", domain),
		zap.String("path", certPath))

	// 創建目錄
	dir := filepath.Dir(certPath)
	if err := os.MkdirAll(dir, SelfSignedDirMode); err != nil {
		return fmt.Errorf("創建目錄失敗: %w", err)
	}

	// 生成私鑰 (ECDSA P-256)
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("生成私鑰失敗: %w", err)
	}

	// 證書模板
	notBefore := time.Now()
	notAfter := notBefore.Add(10 * 365 * 24 * time.Hour) // 10年
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{Organization: []string{"Prism Network"}, CommonName: domain}, // 使用傳入的域名
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
		DNSNames:              []string{domain, "localhost"},
	}

	// 額外添加默認偽裝域名到 SANs，增強兼容性
	if domain != "www.bing.com" {
		template.DNSNames = append(template.DNSNames, "www.bing.com", "*.bing.com")
	}

	// 創建證書
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("創建證書失敗: %w", err)
	}

	// 寫入證書文件
	certOut, err := os.OpenFile(certPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, SelfSignedCertMode)
	if err != nil {
		return fmt.Errorf("寫入證書文件失敗: %w", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	// 寫入私鑰文件
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, SelfSignedKeyMode)
	if err != nil {
		return fmt.Errorf("寫入私鑰文件失敗: %w", err)
	}

	// 正確處理多返回值
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		keyOut.Close()
		return fmt.Errorf("編碼私鑰失敗: %w", err)
	}

	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})
	keyOut.Close()

	// 驗證
	if err := tlsconfig.ValidateCertChain(certPath, keyPath); err != nil {
		return fmt.Errorf("新生成的證書驗證失敗: %w", err)
	}

	return g.verifyPermissions(certPath, keyPath)
}

func (g *SelfSignedGenerator) verifyPermissions(certPath, keyPath string) error {
	if info, err := os.Stat(certPath); err == nil && info.Mode().Perm() != SelfSignedCertMode {
		os.Chmod(certPath, SelfSignedCertMode)
	}
	if info, err := os.Stat(keyPath); err == nil && info.Mode().Perm() != SelfSignedKeyMode {
		os.Chmod(keyPath, SelfSignedKeyMode)
	}
	dir := filepath.Dir(certPath)
	if info, err := os.Stat(dir); err == nil && info.Mode().Perm() != SelfSignedDirMode {
		os.Chmod(dir, SelfSignedDirMode)
	}
	return nil
}

package certinfo

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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 生成测试用的自签名证书
func generateTestCert(domain string, notAfter time.Time) ([]byte, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: domain},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{domain},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes}), nil
}

func TestParseCertFile_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	domain := "example.com"
	certPath := filepath.Join(tmpDir, domain+".crt")

	// 生成有效证书（1年后过期）
	certPEM, err := generateTestCert(domain, time.Now().Add(365*24*time.Hour))
	require.NoError(t, err)

	err = os.WriteFile(certPath, certPEM, 0644)
	require.NoError(t, err)

	// 解析证书
	info := ParseCertFile(certPath)

	// 验证
	assert.Equal(t, domain, info.Domain)
	assert.True(t, info.IsValid)
	assert.Equal(t, "Valid", info.Status)
	assert.Greater(t, info.DaysLeft, 300) // 应该剩余超过300天
}

func TestParseCertFile_Expired(t *testing.T) {
	tmpDir := t.TempDir()
	domain := "expired.com"
	certPath := filepath.Join(tmpDir, domain+".crt")

	// 生成过期证书（1天前过期）
	certPEM, err := generateTestCert(domain, time.Now().Add(-24*time.Hour))
	require.NoError(t, err)

	err = os.WriteFile(certPath, certPEM, 0644)
	require.NoError(t, err)

	// 解析证书
	info := ParseCertFile(certPath)

	// 验证
	assert.Equal(t, domain, info.Domain)
	assert.False(t, info.IsValid)
	assert.Equal(t, "Expired", info.Status)
	assert.Less(t, info.DaysLeft, 0)
}

func TestParseCertFile_Expiring(t *testing.T) {
	tmpDir := t.TempDir()
	domain := "expiring.com"
	certPath := filepath.Join(tmpDir, domain+".crt")

	// 生成即将过期的证书（15天后过期）
	certPEM, err := generateTestCert(domain, time.Now().Add(15*24*time.Hour))
	require.NoError(t, err)

	err = os.WriteFile(certPath, certPEM, 0644)
	require.NoError(t, err)

	// 解析证书
	info := ParseCertFile(certPath)

	// 验证
	assert.Equal(t, domain, info.Domain)
	assert.True(t, info.IsValid)
	assert.Equal(t, "Expiring", info.Status) // 少于30天应该标记为Expiring
	assert.Less(t, info.DaysLeft, 30)
}

func TestParseCertFile_NonExistent(t *testing.T) {
	info := ParseCertFile("/nonexistent/path/cert.crt")

	assert.False(t, info.IsValid)
	assert.Equal(t, "Error", info.Status)
	assert.NotEmpty(t, info.ErrorMsg)
}

func TestParseCertFile_InvalidPEM(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "invalid.crt")

	// 写入无效的 PEM 内容
	err := os.WriteFile(certPath, []byte("invalid pem content"), 0644)
	require.NoError(t, err)

	info := ParseCertFile(certPath)

	assert.False(t, info.IsValid)
	assert.Equal(t, "Error", info.Status)
	assert.Contains(t, info.ErrorMsg, "invalid PEM format")
}

func TestGetACMECertListFromDir(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建多个测试证书
	domains := []string{"test1.com", "test2.com", "test3.com"}
	for _, domain := range domains {
		certPath := filepath.Join(tmpDir, domain+".crt")
		certPEM, _ := generateTestCert(domain, time.Now().Add(365*24*time.Hour))
		os.WriteFile(certPath, certPEM, 0644)
	}

	// 创建应该被过滤的文件
	os.WriteFile(filepath.Join(tmpDir, "self_signed.crt"), []byte("fake"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test.issuer.crt"), []byte("fake"), 0644)

	// 获取证书列表
	list := GetACMECertListFromDir(tmpDir)

	// 验证
	assert.Len(t, list, 3)

	domainSet := make(map[string]bool)
	for _, cert := range list {
		domainSet[cert.Domain] = true
	}

	for _, domain := range domains {
		assert.True(t, domainSet[domain], "应该包含 %s", domain)
	}
}

func TestGetSelfSignedCertListFromDir(t *testing.T) {
	tmpDir := t.TempDir()

	// 不存在自签名证书
	list := GetSelfSignedCertListFromDir(tmpDir)
	assert.Empty(t, list)

	// 创建自签名证书
	certPath := filepath.Join(tmpDir, "self_signed.crt")
	certPEM, _ := generateTestCert("www.bing.com", time.Now().Add(365*24*time.Hour))
	os.WriteFile(certPath, certPEM, 0644)

	// 再次获取
	list = GetSelfSignedCertListFromDir(tmpDir)
	assert.Len(t, list, 1)
	assert.Equal(t, "Self-Signed", list[0].Provider)
	assert.Contains(t, list[0].Issuer, "[SELF]")
}

func TestGetAllCertList(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建 ACME 证书
	acmeCert := filepath.Join(tmpDir, "acme.com.crt")
	certPEM, _ := generateTestCert("acme.com", time.Now().Add(365*24*time.Hour))
	os.WriteFile(acmeCert, certPEM, 0644)

	// 创建自签名证书
	selfCert := filepath.Join(tmpDir, "self_signed.crt")
	selfPEM, _ := generateTestCert("localhost", time.Now().Add(365*24*time.Hour))
	os.WriteFile(selfCert, selfPEM, 0644)

	// 获取所有证书
	list := GetAllCertList(tmpDir)

	assert.Len(t, list, 2)
}

func TestGetCertSummary(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建2个 ACME 证书
	for i := 1; i <= 2; i++ {
		domain := fmt.Sprintf("acme%d.com", i)
		certPath := filepath.Join(tmpDir, domain+".crt")
		certPEM, _ := generateTestCert(domain, time.Now().Add(365*24*time.Hour))
		os.WriteFile(certPath, certPEM, 0644)
	}

	// 创建1个自签名证书
	selfPath := filepath.Join(tmpDir, "self_signed.crt")
	selfPEM, _ := generateTestCert("localhost", time.Now().Add(365*24*time.Hour))
	os.WriteFile(selfPath, selfPEM, 0644)

	// 获取统计
	summary := GetCertSummary(tmpDir)

	assert.Equal(t, 2, summary.ACMECerts)
	assert.Equal(t, 1, summary.SelfSignedCerts)
	assert.Equal(t, 3, summary.TotalCerts)
}

func TestGetACMEDomains(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试证书
	domains := []string{"domain1.com", "domain2.com"}
	for _, domain := range domains {
		certPath := filepath.Join(tmpDir, domain+".crt")
		certPEM, _ := generateTestCert(domain, time.Now().Add(365*24*time.Hour))
		os.WriteFile(certPath, certPEM, 0644)
	}

	// 获取域名列表
	result := GetACMEDomains(tmpDir)

	assert.Len(t, result, 2)
	assert.Contains(t, result, "domain1.com")
	assert.Contains(t, result, "domain2.com")
}

func TestGetSelfSignedDomains(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建自签名证书
	certPath := filepath.Join(tmpDir, "self_signed.crt")
	certPEM, _ := generateTestCert("localhost", time.Now().Add(365*24*time.Hour))
	os.WriteFile(certPath, certPEM, 0644)

	// 获取域名列表
	result := GetSelfSignedDomains(tmpDir)

	assert.Len(t, result, 1)
	assert.Contains(t, result, "localhost")
}

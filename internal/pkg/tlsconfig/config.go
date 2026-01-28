package tlsconfig

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// SecureConfig 返回符合現代安全標準的 TLS 配置
// 參考 Mozilla Modern 兼容性級別
func SecureConfig() *tls.Config {
	return &tls.Config{
		// 強制 TLS 1.2+
		MinVersion: tls.VersionTLS12,
		// 顯式支持 TLS 1.3
		MaxVersion: tls.VersionTLS13,

		// 禁用不安全的加密套件 (主要針對 TLS 1.2)
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},

		// 優先使用服務器端的套件順序
		PreferServerCipherSuites: true,

		// 曲線偏好
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},

		// 關鍵：應用層協議協商 (ALPN)
		// 對於代理服務器，這能確保流量正確識別為 HTTP/2 或 HTTP/1.1
		NextProtos: []string{"h2", "http/1.1"},
	}
}

// ValidateCertChain 驗證證書和私鑰是否匹配且有效
func ValidateCertChain(certPath, keyPath string) error {
	// 1. 加載密鑰對
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("加載密鑰對失敗: %w", err)
	}

	// 2. 解析證書
	if len(cert.Certificate) == 0 {
		return errors.New("證書鏈為空")
	}
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("解析 x509 證書失敗: %w", err)
	}

	// 3. 驗證公鑰與私鑰是否匹配
	switch pub := x509Cert.PublicKey.(type) {
	case *rsa.PublicKey:
		priv, ok := cert.PrivateKey.(*rsa.PrivateKey)
		if !ok {
			return errors.New("私鑰類型不匹配 (預期 RSA)")
		}
		if pub.N.Cmp(priv.N) != 0 || pub.E != priv.E {
			return errors.New("RSA 公鑰與私鑰不匹配")
		}
	case *ecdsa.PublicKey:
		priv, ok := cert.PrivateKey.(*ecdsa.PrivateKey)
		if !ok {
			return errors.New("私鑰類型不匹配 (預期 ECDSA)")
		}
		if pub.X.Cmp(priv.X) != 0 || pub.Y.Cmp(priv.Y) != 0 {
			return errors.New("ECDSA 公鑰與私鑰不匹配")
		}
	default:
		return fmt.Errorf("不支持的公鑰算法: %T", x509Cert.PublicKey)
	}

	return nil
}

// ParseCertFromFile 簡單解析證書文件以獲取域名信息
func ParseCertFromFile(certPath string) (*x509.Certificate, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("無效的 PEM 格式")
	}

	return x509.ParseCertificate(block.Bytes)
}

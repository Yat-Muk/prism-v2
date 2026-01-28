package certinfo

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CertInfo 定義證書的基本信息
type CertInfo struct {
	Domain    string        `json:"domain"`
	Issuer    string        `json:"issuer"`
	NotAfter  time.Time     `json:"not_after"`
	Remaining time.Duration `json:"remaining"`
	IsValid   bool          `json:"is_valid"`
	ErrorMsg  string        `json:"error,omitempty"`

	Provider string `json:"provider"` // 例如: "Let's Encrypt", "Self-Signed"

	ExpiresAt string `json:"expires_at"` // 例如: "2023-12-31T23:59:59Z" (RFC3339)

	ExpireDate string `json:"expire_date"` // 例如: "2023-12-31" (僅用於顯示)
	DaysLeft   int    `json:"days_left"`   // 例如: 89
	Status     string `json:"status"`      // 例如: "Valid", "Expiring", "Expired"
}

// CertSummary 證書統計概況
type CertSummary struct {
	ACMECerts       int `json:"acme_count"`
	SelfSignedCerts int `json:"self_signed_count"`
	TotalCerts      int `json:"total_count"`
}

// ParseCertFile 解析單個證書檔案
func ParseCertFile(path string) CertInfo {
	info := CertInfo{
		IsValid: false,
		Status:  "Error",
	}

	data, err := os.ReadFile(path)
	if err != nil {
		info.ErrorMsg = err.Error()
		return info
	}

	block, _ := pem.Decode(data)
	if block == nil {
		info.ErrorMsg = "invalid PEM format"
		return info
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		info.ErrorMsg = err.Error()
		return info
	}

	// 1. 提取域名
	domain := ""
	if len(cert.DNSNames) > 0 {
		domain = cert.DNSNames[0]
	}
	if domain == "" {
		domain = cert.Subject.CommonName
	}
	info.Domain = domain

	// 2. 提取發行者
	info.Issuer = cert.Issuer.CommonName
	if len(cert.Issuer.Organization) > 0 {
		info.Issuer = cert.Issuer.Organization[0]
	}

	// 3. 計算時間與狀態
	info.NotAfter = cert.NotAfter
	info.Remaining = time.Until(cert.NotAfter)
	info.IsValid = time.Now().Before(cert.NotAfter)

	// 4. 填充顯示用字段
	info.ExpiresAt = cert.NotAfter.Format(time.RFC3339)
	info.ExpireDate = cert.NotAfter.Format("2006-01-02")
	info.DaysLeft = int(info.Remaining.Hours() / 24)

	// 5. 判斷 Provider (簡單歸類)
	if strings.Contains(info.Issuer, "Let's Encrypt") {
		info.Provider = "Let's Encrypt"
	} else if strings.Contains(info.Issuer, "ZeroSSL") {
		info.Provider = "ZeroSSL"
	} else if strings.Contains(info.Issuer, "Google") {
		info.Provider = "Google"
	} else {
		info.Provider = info.Issuer // 默認使用 Issuer 名稱
	}

	// 6. 判斷狀態字符串
	if !info.IsValid {
		info.Status = "Expired"
	} else if info.DaysLeft < 30 {
		info.Status = "Expiring" // 快過期警告
	} else {
		info.Status = "Valid"
	}

	return info
}

// GetACMECertListFromDir 從指定目錄讀取 ACME 證書
func GetACMECertListFromDir(certDir string) []CertInfo {
	var list []CertInfo

	entries, err := os.ReadDir(certDir)
	if err != nil {
		return list
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// 過濾規則：以 .crt 結尾，且不是 issuer 證書，也不是自簽證書
		if strings.HasSuffix(entry.Name(), ".crt") &&
			!strings.HasSuffix(entry.Name(), ".issuer.crt") &&
			entry.Name() != "self_signed.crt" {

			path := filepath.Join(certDir, entry.Name())
			info := ParseCertFile(path)

			// 校驗文件名與證書內容是否匹配
			expectedDomain := strings.TrimSuffix(entry.Name(), ".crt")
			if info.Domain != expectedDomain {
				info.Domain = expectedDomain // 優先使用文件名作為標識
			}
			list = append(list, info)
		}
	}
	return list
}

// GetSelfSignedCertListFromDir 從指定目錄讀取自簽證書
func GetSelfSignedCertListFromDir(certDir string) []CertInfo {
	targetPath := filepath.Join(certDir, "self_signed.crt")

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return []CertInfo{}
	}

	info := ParseCertFile(targetPath)
	if info.IsValid {
		info.Issuer = "[SELF] " + info.Issuer
		// 強制標記為自簽名，方便前端分類
		info.Provider = "Self-Signed"
		info.Status = "Valid"
	}

	return []CertInfo{info}
}

// GetAllCertList 獲取所有證書（ACME + 自簽）
func GetAllCertList(certDir string) []CertInfo {
	acme := GetACMECertListFromDir(certDir)
	self := GetSelfSignedCertListFromDir(certDir)
	return append(acme, self...)
}

// GetACMEDomains 獲取有效的 ACME 域名列表
func GetACMEDomains(certDir string) []string {
	return pickValidDomains(GetACMECertListFromDir(certDir))
}

// GetSelfSignedDomains 獲取有效的自簽域名列表
func GetSelfSignedDomains(certDir string) []string {
	return pickValidDomains(GetSelfSignedCertListFromDir(certDir))
}

// GetCertSummary 獲取證書統計信息
func GetCertSummary(certDir string) CertSummary {
	acme := GetACMECertListFromDir(certDir)
	self := GetSelfSignedCertListFromDir(certDir)
	return CertSummary{
		ACMECerts:       len(acme),
		SelfSignedCerts: len(self),
		TotalCerts:      len(acme) + len(self),
	}
}

// 內部輔助函數：提取有效域名
func pickValidDomains(list []CertInfo) []string {
	var out []string
	for _, c := range list {
		// 這裡可以根據需求決定是否只返回有效證書，目前只檢查域名非空
		if c.Domain != "" {
			out = append(out, c.Domain)
		}
	}
	return out
}

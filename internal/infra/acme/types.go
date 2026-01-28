package acme

import (
	"context"
	"time"

	"github.com/go-acme/lego/v4/certificate"
)

// Provider 定義 ACME 驗證方式
type Provider string

const (
	ProviderCloudflare Provider = "cloudflare"
	ProviderAliDNS     Provider = "alidns"
	ProviderDNSPod     Provider = "dnspod"
	ProviderAWS        Provider = "route53"
	ProviderGCloud     Provider = "gcloud"
	ProviderHTTP       Provider = "http-01" // 必須存在
)

// CertStatus 證書狀態結構
type CertStatus struct {
	Domain    string    `json:"domain"`
	ExpiresAt time.Time `json:"expires_at"`
	DaysLeft  int       `json:"days_left"`
	IsValid   bool      `json:"is_valid"`
	Provider  Provider  `json:"provider"`
	Serial    string    `json:"serial,omitempty"`
}

// Client ACME 客戶端接口
// 這是 application 層依賴的接口定義
type Client interface {
	// ObtainCertificate 申請證書
	ObtainCertificate(ctx context.Context, domain string) (*certificate.Resource, error)
}

package state

import (
	"github.com/Yat-Muk/prism-v2/internal/tui/types"
)

// CertState 證書狀態管理
type CertState struct {
	// 證書列表數據
	ACMEDomains       []string
	SelfSignedDomains []string
	CertList          []types.CertInfo
	CurrentCAProvider string

	// 當前操作上下文
	SelectedDomain    string
	SelectedProvider  string
	DNSProviderID     string
	DNSProviderSecret string

	// DNS 輸入嚮導步驟 (0:域名, 1:ID, 2:Secret)
	DNSStep int

	// 刪除確認
	CertConfirmMode bool
	CertToDelete    string

	// 證書模式切換
	CertModeEnabledProtos []int

	// 用於臨時存儲用戶輸入的郵箱
	ACMEEmail string

	// HTTP 申請步驟 (0: 郵箱, 1: 域名)
	HTTPStep int
}

// NewCertState 構造函數
func NewCertState() *CertState {
	return &CertState{
		ACMEDomains:       []string{},
		SelfSignedDomains: []string{},
		CertList:          []types.CertInfo{},
		DNSStep:           0,
	}
}

// ==========================================
// 業務邏輯方法 (只保留真正有邏輯的操作)
// ==========================================

// NextDNSStep 下一步
func (s *CertState) NextDNSStep() { s.DNSStep++ }

// PrevDNSStep 上一步
func (s *CertState) PrevDNSStep() {
	if s.DNSStep > 0 {
		s.DNSStep--
	}
}

// ResetDNSStep 重置步驟
func (s *CertState) ResetDNSStep() { s.DNSStep = 0 }

// PrepareCertDeletion 準備刪除 (封裝了設置刪除目標和開啟確認模式的原子操作)
func (s *CertState) PrepareCertDeletion(domain string) {
	s.CertToDelete = domain
	s.CertConfirmMode = true
}

// ResetCertDeletion 重置刪除狀態
func (s *CertState) ResetCertDeletion() {
	s.CertToDelete = ""
	s.CertConfirmMode = false
}

// RefreshCertList 快捷更新方法
func (s *CertState) RefreshCertList(acme, self []string, list []types.CertInfo, currentCA string) {
	s.ACMEDomains = acme
	s.SelfSignedDomains = self
	s.CertList = list
	s.CurrentCAProvider = currentCA
}

// 輔助方法
func (s *CertState) ResetHTTPStep() { s.HTTPStep = 0 }
func (s *CertState) NextHTTPStep()  { s.HTTPStep++ }
func (s *CertState) PrevHTTPStep() {
	if s.HTTPStep > 0 {
		s.HTTPStep--
	}
}

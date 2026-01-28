package certinfo

import (
	"os"
)

// Repository 定義證書數據庫的接口
// 專業點：Application 層只依賴這個接口，不依賴具體實現
type Repository interface {
	ListCerts() ([]CertInfo, error)
}

// fileRepository 是基於文件系統的實現
type fileRepository struct {
	certDir string
}

// NewRepository 創建一個新的倉庫實例
func NewRepository(certDir string) Repository {
	return &fileRepository{certDir: certDir}
}

// ListCerts 實現接口方法
func (r *fileRepository) ListCerts() ([]CertInfo, error) {
	// 防禦性編程：如果目錄不存在，視為空列表
	if _, err := os.Stat(r.certDir); os.IsNotExist(err) {
		return []CertInfo{}, nil
	}
	// 複用已有的解析邏輯
	return GetACMECertListFromDir(r.certDir), nil
}

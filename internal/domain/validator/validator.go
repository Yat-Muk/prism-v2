package validator

import (
	"errors"
	"net"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// 預編譯正則表達式，避免在熱路徑中重複編譯
var (
	// TLD 驗證：至少 2 個字母
	reTLD = regexp.MustCompile(`^[a-zA-Z]{2,}$`)
	// Label 驗證：字母、數字、連字號
	reLabel = regexp.MustCompile(`^[a-zA-Z0-9\-]+$`)
	// 文件名驗證：允許字母、數字、點、橫線、下劃線
	reFilename = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

// ValidateDomain 驗證域名格式（用於 SNI / 域名輸入）
func ValidateDomain(domain string) bool {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return false
	}

	// 不允許純 IP
	if net.ParseIP(domain) != nil {
		return false
	}

	// 至少要有一個點
	if !strings.Contains(domain, ".") {
		return false
	}

	// 總長度限制，不能以點開頭或結尾
	if len(domain) > 253 || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}

	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return false
	}

	// 檢查頂級域名
	tld := labels[len(labels)-1]
	if !reTLD.MatchString(tld) {
		return false
	}

	// 每個 label 只能包含字母、數字、連字號；長度 1–63，不能以連字號開頭或結尾
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if !reLabel.MatchString(label) {
			return false
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
	}

	return true
}

// ValidateUUID 驗證 UUID 格式（標準 36 字元形式）
func ValidateUUID(uuidStr string) bool {
	uuidStr = strings.TrimSpace(uuidStr)
	if len(uuidStr) != 36 {
		return false
	}
	_, err := uuid.Parse(uuidStr)
	return err == nil
}

// ValidateFilename 驗證文件名安全性（防止路徑遍歷）
func ValidateFilename(filename string) error {
	filename = strings.TrimSpace(filename)

	if filename == "" {
		return errors.New("文件名不能為空")
	}

	// 1. 檢查路徑遍歷攻擊
	if strings.Contains(filename, "..") {
		return errors.New("文件名不能包含 '..' (路徑遍歷攻擊)")
	}

	// 2. 禁止路徑分隔符 (考慮操作系統兼容性)
	if strings.ContainsAny(filename, `/\`) {
		return errors.New("文件名不能包含路徑分隔符")
	}

	// 3. 禁止空字節注入
	if strings.Contains(filename, "\x00") {
		return errors.New("文件名不能包含空字節")
	}

	// 4. 檢查文件名長度（Linux/Unix: 255 字節上限）
	if len(filename) > 255 {
		return errors.New("文件名過長（最多 255 字符）")
	}

	// 5. 驗證字符合法性
	if !reFilename.MatchString(filename) {
		return errors.New("文件名只能包含字母、數字、點、橫線、下劃線")
	}

	// 6. 禁止特殊文件名 (Windows 保留字)
	reserved := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	filenameUpper := strings.ToUpper(filename)
	// 針對 "." 和 ".." 已在前面檢查，此處主要檢查 Windows 保留字
	// 精確匹配或作為前綴匹配 (例如 CON.txt 也是非法的)
	for _, r := range reserved {
		if filenameUpper == r || strings.HasPrefix(filenameUpper, r+".") {
			return errors.New("文件名為系統保留名稱")
		}
	}

	return nil
}

// ValidateSafePath 驗證完整路徑安全性
// 確保目標路徑在指定的基礎目錄內，防止路徑遍歷到系統敏感目錄
func ValidateSafePath(baseDir, filename string) error {
	// 1. 先驗證文件名
	if err := ValidateFilename(filename); err != nil {
		return err
	}

	// 2. 獲取基礎目錄的絕對路徑
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return errors.New("無法解析基礎目錄: " + err.Error())
	}

	// 3. 構建完整路徑並解析絕對路徑
	// 注意：filepath.Join 會自動處理清理路徑中的 ".."
	fullPath := filepath.Join(absBase, filename)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return errors.New("無法解析路徑: " + err.Error())
	}

	// 4. 確保最終路徑在基礎目錄內 (前綴檢查)
	// 注意：為了正確比較，確保路徑分隔符一致
	if !strings.HasPrefix(absPath, absBase) {
		return errors.New("路徑不在允許的基礎目錄內")
	}

	return nil
}

// ValidateCertDomain 驗證證書域名（結合 ValidateDomain + 路徑安全）
// 用於證書操作場景，既驗證域名格式，也確保不會導致路徑遍歷
func ValidateCertDomain(domain string) error {
	// 1. 基礎格式驗證
	if !ValidateDomain(domain) {
		return errors.New("域名格式無效")
	}

	// 2. 路徑安全驗證（域名會用作文件名）
	if err := ValidateFilename(domain); err != nil {
		return errors.New("域名包含不安全字符: " + err.Error())
	}

	return nil
}

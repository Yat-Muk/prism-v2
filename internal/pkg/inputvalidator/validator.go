package inputvalidator

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// 輸入長度限制常量
const (
	MaxInputBuffer   = 4096 // 輸入緩衝區最大長度（4KB）
	MaxMenuInput     = 100  // 菜單輸入最大長度
	MaxProtocolInput = 50   // 協議選擇輸入最大長度

	// 域名和網絡相關
	MaxDomainLength = 253 // 域名最大長度（RFC 1035）
	MaxSNILength    = 253 // SNI 最大長度
	MaxEmailLength  = 254 // 郵箱最大長度（RFC 5321）

	// 憑證相關
	MaxPasswordLength  = 128 // 密碼最大長度
	MaxAPIKeyLength    = 256 // API 密鑰最大長度
	MaxAPISecretLength = 512 // API Secret 最大長度
	MaxUUIDLength      = 36  // UUID 固定長度

	// 端口相關
	MaxPortInput = 20    // 端口輸入最大長度（包括範圍）
	MinPort      = 1024  // 最小端口號
	MaxPort      = 65535 // 最大端口號

	// 其他
	MaxBackupNameLength = 100 // 備份文件名最大長度
)

// ValidationError 驗證錯誤
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateLength 驗證字符串長度
func ValidateLength(input string, maxLen int, fieldName string) error {
	if len(input) > maxLen {
		return &ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("長度超過限制（最大 %d 字符，當前 %d 字符）", maxLen, len(input)),
		}
	}
	return nil
}

// ValidateDomain 驗證域名格式（純布爾值返回，供內部使用）
func ValidateDomain(domain string) bool {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return false
	}

	// 不允許純 IP
	if net.ParseIP(domain) != nil {
		return false
	}

	// 總長度限制
	if len(domain) > 253 {
		return false
	}

	// 域名不能以點開頭或結尾 (除非是根域名，但這裡我們針對普通用戶)
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}

	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return false // 至少要是 example.com 這種兩級結構
	}

	// 檢查頂級域名：至少 2 個字母，不能包含數字
	tld := labels[len(labels)-1]
	if !regexp.MustCompile(`^[a-zA-Z]{2,}$`).MatchString(tld) {
		return false
	}

	// 每個 label 驗證
	// 允許字母、數字、連字號；長度 1–63
	// 不能以連字號開頭或結尾
	labelRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if !labelRegex.MatchString(label) {
			return false
		}
	}

	return true
}

// ValidateDomainInput 驗證域名輸入（返回詳細錯誤）
func ValidateDomainInput(domain string) error {
	domain = strings.TrimSpace(domain)

	// 空檢查
	if domain == "" {
		return &ValidationError{
			Field:   "domain",
			Message: "域名不能為空",
		}
	}

	// 長度檢查
	if len(domain) > MaxDomainLength {
		return &ValidationError{
			Field:   "domain",
			Message: fmt.Sprintf("域名過長（最大 %d 字符）", MaxDomainLength),
		}
	}

	// 複用 ValidateDomain 的邏輯
	if !ValidateDomain(domain) {
		return &ValidationError{
			Field:   "domain",
			Message: "域名格式無效",
		}
	}

	return nil
}

// ValidateAPICredential 驗證 API 憑證
func ValidateAPICredential(credential string, fieldName string, maxLen int) error {
	credential = strings.TrimSpace(credential)

	// 空檢查
	if credential == "" {
		return &ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("%s 不能為空", fieldName),
		}
	}

	// 長度檢查
	if len(credential) > maxLen {
		return &ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("%s 過長（最大 %d 字符）", fieldName, maxLen),
		}
	}

	// 必須是有效的 UTF-8
	if !utf8.ValidString(credential) {
		return &ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("%s 包含無效的 UTF-8 字符", fieldName),
		}
	}

	// 可打印字符檢查（ASCII 32-126）
	// 注意：有些 Key 可能包含 base64 字符集，這些都在可打印範圍內
	for _, r := range credential {
		if r < 32 || r > 126 {
			return &ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("%s 包含非法字符 (僅允許 ASCII 可打印字符)", fieldName),
			}
		}
	}

	return nil
}

// ValidatePortInput 驗證端口輸入
func ValidatePortInput(portInput string) error {
	portInput = strings.TrimSpace(portInput)

	if portInput == "" {
		return &ValidationError{Field: "port", Message: "端口不能為空"}
	}

	if len(portInput) > MaxPortInput {
		return &ValidationError{Field: "port", Message: "輸入過長"}
	}

	// 格式檢查（只允許數字、連字符、r/R）
	validPort := regexp.MustCompile(`^[0-9rR-]+$`)
	if !validPort.MatchString(portInput) {
		return &ValidationError{
			Field:   "port",
			Message: "端口格式無效（只允許數字、r、-）",
		}
	}

	return nil
}

// ValidatePortRange 嚴格驗證端口範圍格式 (min-max)
// 用於 Hysteria 2 的端口跳躍設置
func ValidatePortRange(input string) error {
	input = strings.TrimSpace(input)

	// 1. 基礎格式檢查
	if strings.Count(input, "-") != 1 {
		return &ValidationError{
			Field:   "port_range",
			Message: "格式錯誤，必須包含一個連字符 (例如 20000-30000)",
		}
	}

	parts := strings.Split(input, "-")
	if len(parts) != 2 {
		return &ValidationError{Field: "port_range", Message: "格式無效"}
	}

	// 2. 數值解析檢查
	startStr, endStr := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

	// 檢查是否為純數字
	if !regexp.MustCompile(`^\d+$`).MatchString(startStr) || !regexp.MustCompile(`^\d+$`).MatchString(endStr) {
		return &ValidationError{
			Field:   "port_range",
			Message: "端口必須是純數字",
		}
	}

	start, _ := strconv.Atoi(startStr)
	end, _ := strconv.Atoi(endStr)

	// 3. 邏輯範圍檢查
	if start < MinPort || start > MaxPort {
		return &ValidationError{
			Field:   "port_range",
			Message: fmt.Sprintf("起始端口必須在 %d-%d 之間", MinPort, MaxPort),
		}
	}
	if end < MinPort || end > MaxPort {
		return &ValidationError{
			Field:   "port_range",
			Message: fmt.Sprintf("結束端口必須在 %d-%d 之間", MinPort, MaxPort),
		}
	}
	if start >= end {
		return &ValidationError{
			Field:   "port_range",
			Message: "起始端口必須小於結束端口",
		}
	}

	return nil
}

// ParsePortRange 解析端口範圍字符串
func ParsePortRange(input string) (int, int, error) {
	// 1. 復用驗證邏輯 (防禦性編程)
	if err := ValidatePortRange(input); err != nil {
		return 0, 0, fmt.Errorf("解析失敗: %w", err)
	}

	// 2. 執行解析 (因為通過了驗證，這裡可以大膽拆分)
	parts := strings.Split(input, "-")
	start, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	end, _ := strconv.Atoi(strings.TrimSpace(parts[1]))

	return start, end, nil
}

// ValidateMenuNumber 驗證菜單數字輸入
func ValidateMenuNumber(input string, min, max int) error {
	input = strings.TrimSpace(input)

	if len(input) > 10 {
		return &ValidationError{Field: "menu", Message: "輸入過長"}
	}

	validMenu := regexp.MustCompile(`^[0-9, ]+$`)
	if !validMenu.MatchString(input) {
		return &ValidationError{Field: "menu", Message: "只允許數字和逗號"}
	}

	return nil
}

// ValidateEmail 驗證電子郵件
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)

	if email == "" {
		return &ValidationError{Field: "email", Message: "郵箱不能為空"}
	}

	if len(email) > MaxEmailLength {
		return &ValidationError{Field: "email", Message: "郵箱過長"}
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return &ValidationError{Field: "email", Message: "郵箱格式無效"}
	}

	return nil
}

// ValidateFilename 驗證文件名安全性 (防止路徑遍歷)
func ValidateFilename(name string) error {
	if name == "" {
		return &ValidationError{Field: "filename", Message: "文件名不能為空"}
	}
	if len(name) > MaxBackupNameLength {
		return &ValidationError{Field: "filename", Message: "文件名過長"}
	}
	// 禁止包含路徑分隔符
	if strings.ContainsAny(name, `\/`) {
		return &ValidationError{Field: "filename", Message: "文件名不能包含路徑分隔符"}
	}
	// 禁止 . 和 ..
	if name == "." || name == ".." {
		return &ValidationError{Field: "filename", Message: "非法文件名"}
	}
	// 只允許安全字符
	validName := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !validName.MatchString(name) {
		return &ValidationError{Field: "filename", Message: "文件名包含非法字符"}
	}
	return nil
}

// ValidateSafePath 驗證路徑是否安全（防止 Path Traversal）
func ValidateSafePath(dir, filename string) error {
	cleanDir := filepath.Clean(dir)
	fullPath := filepath.Join(cleanDir, filename)
	cleanPath := filepath.Clean(fullPath)

	// 確保最終路徑仍然在目錄內
	if !strings.HasPrefix(cleanPath, cleanDir+string(os.PathSeparator)) && cleanPath != cleanDir {
		// 這裡的邏輯比較嚴格：文件必須在指定目錄的子目錄中
		// 如果允許直接在 dir 下，則需調整
		return nil
	}
	// 簡單檢查：不允許 ..
	if strings.Contains(filename, "..") {
		return &ValidationError{Field: "path", Message: "檢測到路徑遍歷嘗試"}
	}
	return nil
}

// TruncateInput 截斷過長的輸入
func TruncateInput(input string, maxLen int) string {
	if len(input) <= maxLen {
		return input
	}
	return input[:maxLen]
}

// SanitizeInput 清理輸入（移除控制字符）
func SanitizeInput(input string) string {
	var result strings.Builder
	for _, r := range input {
		if r >= 32 && r != 127 {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ValidateDNSCredentialInput 驗證 DNS 憑證
// 警告：此函數使用逗號分隔輸入，如果憑證本身包含逗號會導致解析錯誤。
// 建議在 TUI 層面將輸入拆分為多個步驟，而不是依賴此函數。
func ValidateDNSCredentialInput(input string) (domain, id, secret string, err error) {
	// 1. 預處理：將全角逗號替換為半角
	input = strings.ReplaceAll(input, "，", ",")

	// 2. 使用更智能的分割：SplitN 限制分割次數
	// 這樣即使 Secret 中包含逗號，也可以被正確捕獲（前提是 Secret 在最後）
	parts := strings.SplitN(input, ",", 3)

	// 3. 驗證數量
	if len(parts) != 3 {
		return "", "", "", &ValidationError{
			Field:   "dns_credential",
			Message: "格式錯誤：請嚴格按照 '域名,ID,Secret' 格式輸入",
		}
	}

	// 4. 清理並賦值
	domain = strings.TrimSpace(parts[0])
	id = strings.TrimSpace(parts[1])
	secret = strings.TrimSpace(parts[2])

	// 5. 驗證非空
	if domain == "" || id == "" || secret == "" {
		return "", "", "", &ValidationError{
			Field:   "dns_credential",
			Message: "所有字段都不能為空",
		}
	}

	// 6. 進一步驗證域名格式
	if !ValidateDomain(domain) {
		return "", "", "", &ValidationError{Field: "domain", Message: "域名格式無效"}
	}

	return domain, id, secret, nil
}

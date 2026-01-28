package sanitizer

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// 敏感字段關鍵詞 (Fast Path 過濾用)
var sensitiveKeywords = []string{
	"password", "passwd", "secret", "token", "key", "auth", "credential",
	"uuid", "bearer", "private", "salt",
}

// 預編譯正則表達式 (Slow Path 用)
var (
	// UUID: 8-4-4-4-12
	uuidRegex = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	// API Key 常見模式 (sk_..., ghp_...)
	apiKeyRegex = regexp.MustCompile(`(?i)(sk|pk|api|ghp|gho|token)_[a-zA-Z0-9_-]{16,}`)
	// Email
	emailRegex = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
)

// Sanitize 對任意對象進行脫敏處理 (通常用於日誌輸出)
func Sanitize(v interface{}) interface{} {
	// 1. 序列化為字符串
	var s string
	switch val := v.(type) {
	case string:
		s = val
	case []byte:
		s = string(val)
	case error:
		s = val.Error()
	case fmt.Stringer:
		s = val.String()
	default:
		bytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("sanitize_error: %v", err)
		}
		s = string(bytes)
	}

	// 2. Fast Path: 快速檢查是否可能包含敏感數據
	// 如果字符串中連一個敏感關鍵詞都沒有，直接返回，避免正則開銷
	if !mightContainSensitiveData(s) {
		return s
	}

	// 3. Slow Path: 執行正則替換
	return sanitizeString(s)
}

// sanitizeString 執行具體的正則替換
func sanitizeString(s string) string {
	// 脫敏 Email
	s = emailRegex.ReplaceAllStringFunc(s, func(match string) string {
		return Email(match)
	})

	// 脫敏 API Key
	s = apiKeyRegex.ReplaceAllStringFunc(s, func(match string) string {
		return APIKey(match)
	})

	// 脫敏 UUID (可選，視業務需求而定，這裡偏向安全)
	s = uuidRegex.ReplaceAllStringFunc(s, func(match string) string {
		return UUID(match)
	})

	return s
}

// mightContainSensitiveData 快速檢查 (O(N) 字符串搜索)
func mightContainSensitiveData(s string) bool {
	sLower := strings.ToLower(s)
	for _, kw := range sensitiveKeywords {
		if strings.Contains(sLower, kw) {
			return true
		}
	}
	// 額外檢查是否包含 @ 符號 (Email)
	if strings.Contains(s, "@") {
		return true
	}
	return false
}

// Helper Functions for specific types

// String 通用字符串脫敏 (保留首尾)
func String(s string, start, end int) string {
	if len(s) <= start+end {
		return "***"
	}
	return s[:start] + "***" + s[len(s)-end:]
}

// Password 密碼全脫敏
func Password(s string) string {
	if s == "" {
		return ""
	}
	return "***MASKED***"
}

// APIKey API Key 脫敏 (保留前綴)
func APIKey(s string) string {
	if len(s) < 8 {
		return "***"
	}
	return s[:4] + "***" + s[len(s)-4:]
}

// Email 郵箱脫敏
func Email(s string) string {
	at := strings.Index(s, "@")
	if at <= 1 {
		return s
	}
	name := s[:at]
	domain := s[at:]

	maskedName := name
	if len(name) > 2 {
		maskedName = name[:2] + "***"
	} else {
		maskedName = name[:1] + "***"
	}
	return maskedName + domain
}

// UUID 脫敏
func UUID(s string) string {
	if len(s) != 36 {
		return s
	}
	return s[:8] + "-****-****-****-" + s[32:]
}

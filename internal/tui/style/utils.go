package style

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// logEntry 定義 sing-box 日誌結構
type logEntry struct {
	Level   string `json:"level"`
	Type    string `json:"type"`
	Message string `json:"msg"` // sing-box 通常使用 "msg"
	Time    string `json:"time"`
	Ts      string `json:"ts"` // 舊版本或不同配置可能使用 "ts"
}

// ansiRegex 用於去除日誌中原有的顏色代碼
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// BuildColoredLogContent 構建帶顏色的日誌字符串 (支持 JSON 解析)
func BuildColoredLogContent(logs []string) string {
	if len(logs) == 0 {
		return ""
	}

	var sb strings.Builder

	// 預定義樣式
	timeStyle := lipgloss.NewStyle().Foreground(Snow3)
	msgStyle := lipgloss.NewStyle().Foreground(Snow1)

	// 級別樣式
	levelError := lipgloss.NewStyle().Foreground(StatusRed)
	levelWarn := lipgloss.NewStyle().Foreground(StatusYellow)
	levelInfo := lipgloss.NewStyle().Foreground(Aurora2)
	levelDebug := lipgloss.NewStyle().Foreground(Muted)

	for _, line := range logs {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// 1. 嘗試解析 JSON
		var entry logEntry
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			// --- JSON 解析成功 ---

			// 獲取並清理字段 (去除原有的 ANSI 碼)
			level := stripANSI(entry.Level)
			if level == "" {
				level = "INFO"
			}

			msg := stripANSI(entry.Message)
			if msg == "" {
				// 有些日誌把內容放在 type 裡
				msg = stripANSI(entry.Type)
			}

			tsStr := entry.Time
			if tsStr == "" {
				tsStr = entry.Ts
			}

			// 格式化時間 (嘗試縮短顯示)
			displayTime := formatTime(tsStr)

			// 根據級別渲染
			var renderedLevel string
			switch strings.ToUpper(level) {
			case "ERROR", "FATAL", "PANIC":
				renderedLevel = levelError.Render(fmt.Sprintf("[ERROR]"))
			case "WARN", "WARNING":
				renderedLevel = levelWarn.Render(fmt.Sprintf("[WARN]"))
			case "DEBUG":
				renderedLevel = levelDebug.Render(fmt.Sprintf("[DEBUG]"))
			default:
				renderedLevel = levelInfo.Render(fmt.Sprintf("[INFO]"))
			}

			// 拼接: [12:30:59] [INFO] 消息內容
			sb.WriteString(fmt.Sprintf("%s%s%s\n",
				timeStyle.Render(displayTime),
				renderedLevel,
				msgStyle.Render(msg),
			))

		} else {
			// --- JSON 解析失敗 (非 JSON 格式日誌) ---
			// 使用舊的關鍵字匹配邏輯進行降級處理

			// 先清理原始行中的 ANSI 碼，防止干擾
			cleanLine := stripANSI(line)

			if strings.Contains(cleanLine, "ERROR") || strings.Contains(cleanLine, "FATAL") {
				sb.WriteString(ErrorText(cleanLine))
			} else if strings.Contains(cleanLine, "WARN") {
				sb.WriteString(WarningText(cleanLine))
			} else if strings.Contains(cleanLine, "DEBUG") {
				sb.WriteString(MutedText(cleanLine))
			} else {
				sb.WriteString(SnowText(cleanLine))
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// stripANSI 去除字符串中的 ANSI 轉義碼
func stripANSI(str string) string {
	return ansiRegex.ReplaceAllString(str, "")
}

// formatTime 嘗試解析並簡化時間字符串
func formatTime(raw string) string {
	// 嘗試解析 RFC3339 (例如 2026-01-21T22:40:00Z)
	t, err := time.Parse(time.RFC3339, raw)
	if err == nil {
		return fmt.Sprintf("[%s]", t.Format("01-02 15:04:05"))
	}

	// 嘗試解析 Go 默認格式
	t, err = time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", raw)
	if err == nil {
		return fmt.Sprintf("[%s]", t.Format("01-02 15:04:05"))
	}

	// 如果解析失敗或為空，返回原始值或默認佔位
	if len(raw) > 19 {
		return fmt.Sprintf("[%s]", raw[:19]) // 簡單截斷
	}
	if raw == "" {
		return "[Unknown Time]"
	}
	return fmt.Sprintf("[%s]", raw)
}

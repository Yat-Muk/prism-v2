package style

import "github.com/charmbracelet/lipgloss"

// TextColor 返回一個使用指定前景色的 Render 函數。
// 這樣上層可以寫：style.TextColor(style.Info)("保存並應用")
func TextColor(c lipgloss.Color) func(string) string {
	s := lipgloss.NewStyle().Foreground(c)
	return func(str string) string {
		return s.Render(str)
	}
}

// 語義著色快捷函數 --------------------------------------------------

// InfoText 使用 Info 顏色（信息藍）顯示文字。
func InfoText(s string) string {
	return TextColor(Info)(s)
}

// SuccessText 使用 Success 顏色（成功綠）顯示文字。
func SuccessText(s string) string {
	return TextColor(Success)(s)
}

// WarningText 使用 Warning 顏色（警告黃）顯示文字。
func WarningText(s string) string {
	return TextColor(Warning)(s)
}

// ErrorText 使用 Error 顏色（錯誤紅）顯示文字。
func ErrorText(s string) string {
	return TextColor(Error)(s)
}

// MutedText 使用 Muted 顏色（弱化灰）顯示文字。
func MutedText(s string) string {
	return TextColor(Muted)(s)
}

// PrimaryText 使用 Primary 顏色（主題藍）顯示文字。
func PrimaryText(s string) string {
	return TextColor(Primary)(s)
}

// SnowText 主文字顏色（白）。
func SnowText(s string) string {
	return TextColor(Snow1)(s)
}

// VisualLength 計算字符串在等寬終端中的可視寬度。
// CJK 字符按寬度 2，其餘按 1，用於對齊表格與菜單。
func VisualLength(s string) int {
	length := 0
	for _, r := range s {
		// 控制字符不計入寬度
		if r <= 0x1F || (r >= 0x7F && r <= 0x9F) {
			continue
		}
		// 粗略判斷 CJK 統一表意文字區段，算寬 2
		if r >= 0x2E80 && r <= 0x9FFF {
			length += 2
		} else {
			length++
		}
	}
	return length
}

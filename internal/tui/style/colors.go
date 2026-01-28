package style

import "github.com/charmbracelet/lipgloss"

// 原項目配色方案
var (
	// 主色調 - 鮮艷現代
	FutureGreen = lipgloss.Color("#B2FF00") // 螢光綠 - 成功/運行中
	SkyBlue     = lipgloss.Color("#1AAEFC") // 天藍 - 主要強調/Logo
	Violet      = lipgloss.Color("#DDAAFF") // 紫羅蘭 - 次要強調
	Yellow      = lipgloss.Color("#FFDC65") // 明黃 - 警告
	Orange      = lipgloss.Color("#FC7B00") // 橙色 - 中等警告
	Red         = lipgloss.Color("#FF007F") // 紅色 - 錯誤/停止

	// 文字顏色
	White    = lipgloss.Color("#F3F3F0") // 純白 - 主要文字
	Gray     = lipgloss.Color("#C0C0C0") // 淺灰 - 次要文字
	DarkGray = lipgloss.Color("#8A8783") // 深灰 - 弱化文字

	// 背景色（保持暗色背景以突出鮮艷色彩）
	BgDark   = lipgloss.Color("#1a1a1a") // 深黑背景
	BgMedium = lipgloss.Color("#2a2a2a") // 中等背景
	BgLight  = lipgloss.Color("#3a3a3a") // 淺色背景
)

// 功能顏色映射（兼容舊代碼）
var (
	// 主題色
	Primary   = SkyBlue // 主色 - Logo、重要元素
	Secondary = Violet  // 次色 - 次要強調
	Text      = White   // 主要文字

	// 狀態色
	StatusGreen  = FutureGreen // 成功/運行
	StatusYellow = Yellow      // 警告
	StatusOrange = Orange      // 中等警告
	StatusRed    = Red         // 錯誤/停止

	// Aurora 系列（主題色）
	Aurora1 = FutureGreen // 主色
	Aurora2 = SkyBlue     // 次色
	Aurora3 = Violet      // 強調色
	Aurora4 = Orange      // 輔助色

	// Snow 系列（文字）
	Snow1 = White    // 主要文字
	Snow2 = Gray     // 次要文字
	Snow3 = DarkGray // 弱化文字

	// Polar 系列（背景）
	Polar1 = BgDark   // 最深背景
	Polar2 = BgMedium // 中等背景
	Polar3 = BgLight  // 淺色背景
	Polar4 = DarkGray // 邊框/分隔線

	// 其他
	Muted   = DarkGray    // 弱化文字
	Success = FutureGreen // 成功
	Error   = Red         // 錯誤
	Warning = Yellow      // 警告
	Info    = SkyBlue     // 信息
)

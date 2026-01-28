package handlers

import (
	"testing"
	"time"
)

// 測試 formatBytes 函數 (CommandBuilder 內部的私有函數)
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"Zero", 0, "0 B"},
		{"Bytes", 500, "500 B"},
		{"Exact 1KB", 1024, "1.00 KB"}, // 關鍵邊界測試
		{"1.5KB", 1536, "1.50 KB"},
		{"Exact 1MB", 1048576, "1.00 MB"},
		{"Large GB", 10737418240, "10.00 GB"}, // 10 GB
		{"Negative", -1024, "0 B"},            // 負數保護
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.input)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

// 測試 formatDuration 函數
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{"Seconds", 45 * time.Second, "45秒"},
		{"Minutes", 5 * time.Minute, "5分鐘"},
		{"HoursMins", 1*time.Hour + 30*time.Minute, "1小時30分鐘"},
		{"DaysHours", 26 * time.Hour, "1天2小時"},
		{"Zero", 0, "0秒"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.input)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

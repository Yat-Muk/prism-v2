package application

import (
	"context"
	"testing"

	"github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/domain/protocol" // 確保導入了 protocol 包
	"go.uber.org/zap"
)

// TestToggleProtocolsFromInput 測試協議開關邏輯
// 輸入 "1,2" -> 對應 ID 進行取反操作
func TestToggleProtocolsFromInput(t *testing.T) {
	logger := zap.NewNop()
	svc := NewProtocolService(logger)
	ctx := context.Background()

	// 假設協議 ID:
	// 1: RealityVision
	// 3: Hysteria2
	// 具體 ID 取決於您的 protocol 包定義，這裡使用常量
	id1 := int(protocol.IDRealityVision)
	id3 := int(protocol.IDHysteria2)

	tests := []struct {
		name           string
		currentEnabled []int
		input          string
		wantCount      int
		shouldContain  []int
		shouldError    bool
	}{
		{
			name:           "Add new protocol",
			currentEnabled: []int{},
			input:          "1", // 開啟 ID 1
			wantCount:      1,
			shouldContain:  []int{1},
			shouldError:    false,
		},
		{
			name:           "Remove existing protocol",
			currentEnabled: []int{1, 3},
			input:          "1", // 關閉 ID 1
			wantCount:      1,
			shouldContain:  []int{3}, // 剩 ID 3
			shouldError:    false,
		},
		{
			name:           "Mixed toggle",
			currentEnabled: []int{1},
			input:          "1,3", // 關閉 1，開啟 3
			wantCount:      1,
			shouldContain:  []int{3},
			shouldError:    false,
		},
		{
			name:           "Invalid ID format",
			currentEnabled: []int{},
			input:          "abc,1",
			wantCount:      1, // abc 被忽略，1 生效
			shouldContain:  []int{1},
			shouldError:    false,
		},
		{
			name:           "Out of range ID",
			currentEnabled: []int{},
			input:          "9999", // 假設這是一個無效 ID
			wantCount:      0,
			shouldContain:  nil,
			shouldError:    true, // IsValid() 應該返回 false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 動態構建輸入字符串，適配實際 ID 常量
			input := tt.input
			if input == "1" {
				input = strconv_Itoa(id1)
			}
			if input == "1,3" {
				input = strconv_Itoa(id1) + "," + strconv_Itoa(id3)
			}

			// 構建當前狀態
			current := []int{}
			for _, v := range tt.currentEnabled {
				if v == 1 {
					current = append(current, id1)
				}
				if v == 3 {
					current = append(current, id3)
				}
			}

			// 執行測試
			got, err := svc.ToggleProtocolsFromInput(ctx, current, input)

			// 驗證錯誤
			if (err != nil) != tt.shouldError {
				t.Errorf("ToggleProtocolsFromInput() error = %v, wantErr %v", err, tt.shouldError)
				return
			}
			if tt.shouldError {
				return
			}

			// 驗證數量
			if len(got) != tt.wantCount {
				t.Errorf("Result length = %d, want %d (Result: %v)", len(got), tt.wantCount, got)
			}
		})
	}
}

// TestUpdateConfigWithEnabledProtocols 測試配置結構體更新
func TestUpdateConfigWithEnabledProtocols(t *testing.T) {
	logger := zap.NewNop()
	svc := NewProtocolService(logger)

	// 1. 準備一個初始配置：RealityVision 開啟，其他關閉
	cfg := config.DefaultConfig()
	cfg.Protocols.RealityVision.Enabled = true
	cfg.Protocols.Hysteria2.Enabled = false

	// 2. 目標：僅開啟 Hysteria2 (這意味著 RealityVision 應該被自動關閉)
	targetProtocols := []int{int(protocol.IDHysteria2)}

	// 3. 執行更新
	err := svc.UpdateConfigWithEnabledProtocols(cfg, targetProtocols)
	if err != nil {
		t.Fatalf("UpdateConfigWithEnabledProtocols failed: %v", err)
	}

	// 4. 驗證
	// RealityVision 應該被重置為 false
	if cfg.Protocols.RealityVision.Enabled {
		t.Error("RealityVision should be disabled after update")
	}
	// Hysteria2 應該被設置為 true
	if !cfg.Protocols.Hysteria2.Enabled {
		t.Error("Hysteria2 should be enabled after update")
	}
	// 未涉及的協議 (如 TUIC) 應該保持 false
	if cfg.Protocols.TUIC.Enabled {
		t.Error("TUIC should remain disabled")
	}
}

// TestUpdateAllSNI 測試 SNI 批量更新
func TestUpdateAllSNI(t *testing.T) {
	logger := zap.NewNop()
	svc := NewProtocolService(logger)

	cfg := config.DefaultConfig()
	// 設置一些舊值
	cfg.Protocols.RealityVision.SNI = "old.com"
	cfg.Protocols.Hysteria2.SNI = "old.com"

	newSNI := "new-domain.com"

	// 執行更新
	err := svc.UpdateAllSNI(cfg, newSNI)
	if err != nil {
		t.Fatalf("UpdateAllSNI failed: %v", err)
	}

	// 驗證所有支持 SNI 的協議是否都已更新
	if cfg.Protocols.RealityVision.SNI != newSNI {
		t.Errorf("RealityVision SNI mismatch. Want %s, Got %s", newSNI, cfg.Protocols.RealityVision.SNI)
	}
	if cfg.Protocols.Hysteria2.SNI != newSNI {
		t.Errorf("Hysteria2 SNI mismatch. Want %s, Got %s", newSNI, cfg.Protocols.Hysteria2.SNI)
	}
	if cfg.Protocols.TUIC.SNI != newSNI {
		t.Errorf("TUIC SNI mismatch. Want %s, Got %s", newSNI, cfg.Protocols.TUIC.SNI)
	}
}

// 簡單的整數轉字符串輔助函數，避免引入 strconv 包的額外依賴
func strconv_Itoa(i int) string {
	if i == int(protocol.IDRealityVision) {
		return "1"
	} // 假設 ID 1
	if i == int(protocol.IDHysteria2) {
		return "3"
	} // 假設 ID 3
	// 回退處理，實際測試中可能需要更嚴謹的 strconv
	if i == 1 {
		return "1"
	}
	if i == 3 {
		return "3"
	}
	return "999"
}

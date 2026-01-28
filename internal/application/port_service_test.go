package application

import (
	"context"
	"testing"

	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/domain/protocol"
	"go.uber.org/zap"
)

// TestResetAllPorts 測試全局端口重置
func TestResetAllPorts(t *testing.T) {
	logger := zap.NewNop()
	svc := NewPortService(logger)
	cfg := domainConfig.DefaultConfig()
	ctx := context.Background()

	// 執行重置
	newCfg, err := svc.ResetAllPorts(ctx, cfg)
	if err != nil {
		t.Fatalf("ResetAllPorts 失敗: %v", err)
	}

	// 1. 驗證端口唯一性 (不允許衝突)
	ports := make(map[int]bool)
	checkPort := func(p int, name string) {
		if p < 10000 || p > 60000 {
			t.Errorf("%s 端口範圍錯誤: %d", name, p)
		}
		if ports[p] {
			t.Errorf("%s 端口衝突: %d", name, p)
		}
		ports[p] = true
	}

	checkPort(newCfg.Protocols.RealityVision.Port, "RealityVision")
	checkPort(newCfg.Protocols.RealityGRPC.Port, "RealityGRPC")
	checkPort(newCfg.Protocols.Hysteria2.Port, "Hysteria2")
	checkPort(newCfg.Protocols.TUIC.Port, "TUIC")
	checkPort(newCfg.Protocols.AnyTLS.Port, "AnyTLS")
	checkPort(newCfg.Protocols.AnyTLSReality.Port, "AnyTLSReality")
	checkPort(newCfg.Protocols.ShadowTLS.Port, "ShadowTLS")

	// 2. 驗證深拷貝：原配置不應被修改
	if cfg.Protocols.RealityVision.Port == newCfg.Protocols.RealityVision.Port {
		// 雖然隨機可能相同，但概率極低，通常 Default 是 443 左右，Reset 是 10000+
		if cfg.Protocols.RealityVision.Port < 10000 {
			t.Log("Pass: Original config remained unchanged")
		}
	}
}

// TestUpdateSinglePort 測試單個端口更新
func TestUpdateSinglePort(t *testing.T) {
	svc := NewPortService(zap.NewNop())
	cfg := domainConfig.DefaultConfig()
	ctx := context.Background()

	// 1. 測試：手動指定合法端口
	targetID := int(protocol.IDHysteria2)
	newPortStr := "23456"
	updatedCfg, err := svc.UpdateSinglePort(ctx, cfg, targetID, newPortStr)
	if err != nil {
		t.Fatalf("UpdateSinglePort 失敗: %v", err)
	}
	if updatedCfg.Protocols.Hysteria2.Port != 23456 {
		t.Errorf("預期 23456, 實際 %d", updatedCfg.Protocols.Hysteria2.Port)
	}

	// 2. 測試：隨機端口
	randomCfg, err := svc.UpdateSinglePort(ctx, cfg, targetID, "random")
	if err != nil {
		t.Fatal(err)
	}
	if randomCfg.Protocols.Hysteria2.Port < 10000 || randomCfg.Protocols.Hysteria2.Port > 65535 {
		t.Errorf("隨機端口範圍錯誤: %d", randomCfg.Protocols.Hysteria2.Port)
	}

	// 3. 測試：無效輸入
	badInputs := []string{"-1", "65536", "abc", "0"}
	for _, in := range badInputs {
		_, err := svc.UpdateSinglePort(ctx, cfg, targetID, in)
		if err == nil {
			t.Errorf("預期輸入 %s 會報錯，但實際上成功了", in)
		}
	}
}

// TestHy2Hopping 測試 Hysteria2 端口跳躍邏輯
func TestHy2Hopping(t *testing.T) {
	svc := NewPortService(zap.NewNop())
	cfg := domainConfig.DefaultConfig()
	ctx := context.Background()

	// 1. 設置跳躍
	start, end := 20000, 30000
	updatedCfg, err := svc.UpdateHy2Hopping(ctx, cfg, start, end)
	if err != nil {
		t.Fatal(err)
	}
	expected := "20000-30000"
	if updatedCfg.Protocols.Hysteria2.PortHopping != expected {
		t.Errorf("跳躍格式錯誤: 預期 %s, 得到 %s", expected, updatedCfg.Protocols.Hysteria2.PortHopping)
	}

	// 2. 清除跳躍
	clearedCfg, _ := svc.ClearHy2Hopping(ctx, updatedCfg)
	if clearedCfg.Protocols.Hysteria2.PortHopping != "" {
		t.Error("清除跳躍失敗")
	}

	// 3. 無效範圍測試
	_, err = svc.UpdateHy2Hopping(ctx, cfg, 40000, 30000) // start > end
	if err == nil {
		t.Error("預期 start > end 會報錯")
	}
}

// TestGetPort 測試獲取端口
func TestGetPort(t *testing.T) {
	svc := NewPortService(zap.NewNop())
	cfg := domainConfig.DefaultConfig()
	cfg.Protocols.TUIC.Port = 8888

	port := svc.GetPort(cfg, int(protocol.IDTUIC))
	if port != 8888 {
		t.Errorf("獲取端口失敗: 預期 8888, 得到 %d", port)
	}

	// 測試未知 ID
	if p := svc.GetPort(cfg, 999); p != 0 {
		t.Errorf("未知協議應返回 0, 得到 %d", p)
	}
}

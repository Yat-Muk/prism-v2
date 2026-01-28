package protocol

// ID 定義協議的唯一標識符類型
type ID int

const (
	IDNone          ID = 0
	IDRealityVision ID = 1
	IDRealityGRPC   ID = 2
	IDHysteria2     ID = 3
	IDTUIC          ID = 4
	IDAnyTLS        ID = 5
	IDAnyTLSReality ID = 6
	IDShadowTLS     ID = 7
)

// String 實現 Stringer 接口，用於日誌和顯示
func (id ID) String() string {
	switch id {
	case IDRealityVision:
		return "VLESS Reality Vision"
	case IDRealityGRPC:
		return "VLESS Reality gRPC"
	case IDHysteria2:
		return "Hysteria 2"
	case IDTUIC:
		return "TUIC v5"
	case IDAnyTLS:
		return "AnyTLS"
	case IDAnyTLSReality:
		return "AnyTLS Reality"
	case IDShadowTLS:
		return "ShadowTLS v3"
	default:
		return "Unknown"
	}
}

// IsValid 檢查 ID 是否有效
func (id ID) IsValid() bool {
	return id >= IDRealityVision && id <= IDShadowTLS
}

// AllIDs 返回所有支持的協議 ID 列表 (用於遍歷)
func AllIDs() []ID {
	return []ID{
		IDRealityVision,
		IDRealityGRPC,
		IDHysteria2,
		IDTUIC,
		IDAnyTLS,
		IDAnyTLSReality,
		IDShadowTLS,
	}
}

func (id ID) Tag() string {
	switch id {
	case IDRealityVision:
		return "[推薦]"
	case IDHysteria2:
		return "[高速]"
	case IDAnyTLSReality:
		return "[隧道]"
	case IDTUIC:
		return "[極客]"
	default:
		return ""
	}
}

// 用於顯示詳細說明
func (id ID) Description() string {
	switch id {
	case IDRealityVision:
		return "適合大多數網絡環境，穩定性高"
	case IDHysteria2:
		return "基於 UDP，適合惡劣網絡環境搶佔帶寬"
	default:
		return ""
	}
}

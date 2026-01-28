package state

// PortState 端口編輯狀態管理
type PortState struct {
	// 端口編輯交互狀態
	PortEditingProtocol int  // 當前正在編輯端口的協議 ID
	PortEditingMode     bool // 是否處於端口編輯模式
	Hy2EditingHopping   bool // 是否正在編輯 Hysteria 2 跳躍端口

	// 當前端口數據 (Data)
	CurrentPorts map[int]int

	// 專門存儲 Hysteria 2 的跳躍端口範圍字符串 (如 "20000-30000")
	// 因為這不是標準端口，需要單獨字段
	Hy2HoppingRange string
}

// NewPortState 創建端口狀態管理器
func NewPortState() *PortState {
	return &PortState{
		CurrentPorts:    make(map[int]int), // 初始化 int map
		Hy2HoppingRange: "",                // 初始化為空
	}
}

// 業務邏輯方法
func (s *PortState) StartPortEdit(protoID int) {
	s.PortEditingProtocol = protoID
	s.PortEditingMode = true
	s.Hy2EditingHopping = false
}

func (s *PortState) StartHy2HoppingEdit() {
	s.PortEditingProtocol = 3 // Hysteria 2 的 ID
	s.PortEditingMode = true
	s.Hy2EditingHopping = true
}

func (s *PortState) CancelPortEdit() {
	s.PortEditingMode = false
	s.Hy2EditingHopping = false
	s.PortEditingProtocol = 0
}

func (s *PortState) IsEditingProtocol(protoID int) bool {
	return s.PortEditingMode && s.PortEditingProtocol == protoID
}

func (s *PortState) ClearPorts() {
	s.CurrentPorts = make(map[int]int)
	s.Hy2HoppingRange = ""
}

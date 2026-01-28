package state

import (
	"sort"

	"github.com/Yat-Muk/prism-v2/internal/tui/types"
)

// BackupState 管理 TUI 中的備份相關狀態與操作
type BackupState struct {
	BackupList       []types.BackupItem
	SelectedBackup   string
	ConfirmMode      bool // 是否正在確認(YES/NO)
	OperationType    string
	BackupInProgress bool
	SelectingIndex   bool // 是否正在輸入序號
	BackupListCursor int
	PendingOp        string // 用於在確認模式下區分是用戶想恢復還是想刪除
}

// NewBackupState 創建備份狀態
func NewBackupState() *BackupState {
	return &BackupState{
		BackupList: []types.BackupItem{},
		PendingOp:  "restore", // 默認操作為恢復，防止意外
	}
}

// SetBackupList 設置並排序備份列表
func (s *BackupState) SetBackupList(list []types.BackupItem) {
	s.BackupList = append([]types.BackupItem(nil), list...)
	s.sortBackupList()
}

// sortBackupList 按時間倒序排序 (最新的在最前)
func (s *BackupState) sortBackupList() {
	sort.Slice(s.BackupList, func(i, j int) bool {
		return s.BackupList[i].ModTime.After(s.BackupList[j].ModTime)
	})
}

// GetBackupByIndex 安全獲取備份對象
func (s *BackupState) GetBackupByIndex(idx int) (types.BackupItem, bool) {
	if idx < 0 || idx >= len(s.BackupList) {
		return types.BackupItem{}, false
	}
	return s.BackupList[idx], true
}

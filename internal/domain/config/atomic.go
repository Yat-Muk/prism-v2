package config

import (
	"sync"
	"sync/atomic"
)

// AtomicContainer 使用 Atomic Value 實現的配置容器
// 優點：讀取操作 (Get) 是無鎖且零拷貝的，極大減輕 GC 壓力
type AtomicContainer struct {
	store atomic.Value // 存儲 *Config 指針
	mu    sync.Mutex   // 僅用於序列化 Update 操作（寫鎖）
}

// NewAtomicContainer 初始化
func NewAtomicContainer(cfg *Config) *AtomicContainer {
	c := &AtomicContainer{}
	// 初始化時存儲一份深拷貝，確保起點安全
	c.store.Store(cfg.DeepCopy())
	return c
}

// Get 獲取當前配置的快照 (Read)
// 性能：極快，O(1)，無鎖，無內存分配
// ⚠️ 注意：返回的指針指向的內存是只讀的 (Immutable)，絕對不要修改它！
// 修改必須通過 Update 方法進行。
func (c *AtomicContainer) Get() *Config {
	return c.store.Load().(*Config)
}

// Update 原子更新配置 (Write)
// 邏輯：複製舊配置 -> 修改副本 -> 原子替換 -> 舊配置由 GC 回收
func (c *AtomicContainer) Update(fn func(*Config) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. 獲取當前配置（舊）
	oldCfg := c.store.Load().(*Config)

	// 2. 核心：寫時複製 (Copy-On-Write)
	// 我們只在寫入時付出 DeepCopy 的代價
	newCfg := oldCfg.DeepCopy()

	// 3. 在副本上應用修改
	if err := fn(newCfg); err != nil {
		return err
	}

	// 4. 驗證新配置
	if err := newCfg.Validate(); err != nil {
		return err
	}

	// 5. 原子替換指針
	// 這一瞬間，所有新的 Get() 調用都會拿到 newCfg
	// 正在使用 oldCfg 的協程不受影響，繼續使用舊內存，直到它們釋放引用
	c.store.Store(newCfg)

	return nil
}

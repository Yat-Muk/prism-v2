package types

import "time"

// --- Fail2Ban ---
type Fail2BanInfo struct {
	Installed   bool
	Running     bool
	BannedIPs   int
	SSHAttempts int
	MaxRetry    int
	BanTime     string
}

// --- Swap ---
type SwapInfo struct {
	Enabled  bool
	Total    string
	Used     string
	Free     string
	SwapFile string
}

// --- Streaming ---
type StreamingCheckResult struct {
	IPv4     string
	IPv6     string
	Location string
	Netflix  string
	Disney   string
	YouTube  string
	ChatGPT  string
	TikTok   string
}

// --- Cleanup ---
type CleanupInfo struct {
	LogSize   string
	CacheSize string
	TempSize  string
	TotalSize string
}

// --- BBR ---
type BBRInfo struct {
	Enabled       bool
	Type          string
	KernelVersion string
	Algorithm     string
}

// --- Service Health ---
type HealthCheckResult struct {
	OverallStatus   string
	ServiceRunning  bool
	ConfigValid     bool
	Issues          []string
	Recommendations []string
}

// --- Node Info ---
type NodeInfo struct {
	ServerIP        string
	Protocols       []string
	SubscriptionURL string
	QRCodeAvailable bool
}

type ProtocolLink struct {
	Name string
	URL  string
	Port int
}

type SubscriptionInfo struct {
	OnlineURL  string
	OfflineURL string
	UpdateTime string
	NodeCount  int
}

type ClientConfigInfo struct {
	Format     string
	FilePath   string
	FileSize   string
	ConfigType string
	Protocols  []string
}

// LogInfo 日誌信息
type LogInfo struct {
	LogLevel   string
	LogPath    string
	LogSize    string
	TodayLines int
	ErrorCount int
	RecentLogs []string
}

// --- Certificate Info ---
type CertInfo struct {
	Domain     string
	Provider   string
	ExpireDate string
	DaysLeft   int
	Status     string
}

// --- Backup Info ---
type BackupItem struct {
	Name      string
	Path      string
	ModTime   time.Time
	Size      int64
	Encrypted bool
	Verified  bool
}

// --- System Info ---
type SystemStats struct {
	Hostname     string
	OS           string
	Arch         string
	Kernel       string
	Uptime       string
	LoadAvg      string
	CPUModel     string
	CPUUsage     float64
	MemTotal     string
	MemUsed      string
	MemUsage     float64 // 內存百分比
	DiskTotal    string
	DiskUsed     string
	DiskUsage    float64 // 磁盤百分比
	NetSentTotal string
	NetRecvTotal string
	NetSentRate  string // 格式化後的上傳速度 (如 "1.2 MB/s")
	NetRecvRate  string // 格式化後的下載速度
	NetworkTX    float64
	NetworkRX    float64
	BBR          string
	IPv4         string
	IPv6         string
}

type ServiceStats struct {
	Status      string
	Uptime      string
	MemoryUsage string // 原 Memory
	CPUUsage    float64
}

// --- Uninstall ---
type UninstallInfo struct {
	ConfirmStep    int
	KeepConfig     bool
	KeepCerts      bool
	KeepBackups    bool
	KeepLogs       bool
	ConfigPath     string
	CertDir        string
	BackupDir      string
	LogDir         string
	CoreInstalled  bool
	CorePath       string
	ServiceRunning bool
	ConfigExists   bool
	CertsCount     int
	BackupsCount   int
	LogSize        string
	TotalSize      string
}

type UninstallStep struct {
	Name    string
	Status  string
	Message string
}

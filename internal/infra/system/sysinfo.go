package system

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SystemInfo 系統信息採集器
type SystemInfo struct {
	log           *zap.Logger
	lastCPUTime   CPUTime
	lastNetworkIO NetworkIO
	startTime     time.Time
	cachedOSName  string

	// 緩存公網 IP
	publicIPv4 string
	publicIPv6 string
	ipMutex    sync.RWMutex // 讀寫鎖保護 IP 字段
}

// CPUTime CPU 時間
type CPUTime struct {
	User, Nice, System, Idle, IOWait, Irq, SoftIrq, Steal, Guest, GuestNice uint64
}

// NetworkIO 網絡IO
type NetworkIO struct {
	RxBytes uint64
	TxBytes uint64
	Time    time.Time
}

// Stats 系統統計信息
type Stats struct {
	Hostname     string        // 主機名
	OS           string        // 操作系統名稱
	Kernel       string        // 內核版本
	Arch         string        // 架構
	CPUModel     string        // CPU 型號
	LoadAvg      string        // 負載
	CPUUsage     float64       // CPU 使用率 %
	MemoryUsage  float64       // 內存使用率 %
	MemoryTotal  uint64        // 內存總量 (Bytes)
	MemoryUsed   uint64        // 內存使用量 (Bytes)
	DiskUsage    float64       // 磁盤使用率 %
	DiskTotal    uint64        // 磁盤總量 (Bytes)
	DiskUsed     uint64        // 磁盤使用量 (Bytes)
	NetSentTotal uint64        // 總發送流量 (Bytes)
	NetRecvTotal uint64        // 總接收流量 (Bytes)
	NetworkTX    float64       // 實時上傳速度 (MB/s)
	NetworkRX    float64       // 實時下載速度 (MB/s)
	Uptime       time.Duration // 運行時間
	BBR          string        // BBR 狀態
	IPv4         string        // IPv4 地址 (公網)
	IPv6         string        // IPv6 地址 (公網)
}

// ServiceStats 服務統計信息
type ServiceStats struct {
	Status      string
	PID         int
	Memory      string
	MemoryBytes uint64
	Uptime      string
	UptimeDur   time.Duration
}

// NewSystemInfo 初始化
func NewSystemInfo(log *zap.Logger) *SystemInfo {
	s := &SystemInfo{
		log:       log,
		startTime: time.Now(),
	}
	s.initOSName()

	// 初始化第一次讀數
	if cpu, err := s.readCPUTime(); err == nil {
		s.lastCPUTime = cpu
	}
	if netIO, err := s.readNetworkIO(); err == nil {
		s.lastNetworkIO = netIO
	}

	// 啟動後台協程獲取公網 IP (不阻塞主線程)
	go s.refreshPublicIPs()

	return s
}

func (s *SystemInfo) initOSName() {
	s.cachedOSName = "Linux"
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				s.cachedOSName = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
				return
			}
		}
	}
}

// refreshPublicIPs 在後台獲取公網 IP
func (s *SystemInfo) refreshPublicIPs() {
	var wg sync.WaitGroup
	wg.Add(2)

	// 獲取 IPv4
	go func() {
		defer wg.Done()
		// 使用 4.ipw.cn 或 api.ipify.org
		ip := s.fetchIPFromAPI("https://4.ipw.cn")
		if ip != "" {
			s.ipMutex.Lock()
			s.publicIPv4 = ip
			s.ipMutex.Unlock()
		} else {
			// 如果 API 失敗，嘗試回退到命令獲取 (雖然可能是內網 IP，總比沒有好)
			s.ipMutex.Lock()
			s.publicIPv4 = s.getIPv4FromCmd()
			s.ipMutex.Unlock()
		}
	}()

	// 獲取 IPv6
	go func() {
		defer wg.Done()
		ip := s.fetchIPFromAPI("https://6.ipw.cn")
		if ip != "" {
			s.ipMutex.Lock()
			s.publicIPv6 = ip
			s.ipMutex.Unlock()
		} else {
			s.ipMutex.Lock()
			s.publicIPv6 = s.getIPv6FromCmd()
			s.ipMutex.Unlock()
		}
	}()

	wg.Wait()
}

// fetchIPFromAPI 通用 HTTP 獲取 IP 函數
func (s *SystemInfo) fetchIPFromAPI(url string) string {
	client := &http.Client{
		Timeout: 5 * time.Second, // 設置超時防止卡死
	}
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

// GetStats 獲取系統統計信息 (核心方法)
func (s *SystemInfo) GetStats() (*Stats, error) {
	stats := &Stats{
		OS:     s.cachedOSName,
		Arch:   runtime.GOARCH,
		Uptime: time.Since(s.startTime),
	}

	// 1. 基礎信息
	stats.Hostname, _ = os.Hostname()
	stats.CPUModel = s.getCPUModel()
	stats.LoadAvg = s.getLoadAvg()

	// 2. 內核
	if data, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		stats.Kernel = strings.TrimSpace(string(data))
	} else {
		stats.Kernel = "Unknown"
	}

	// 3. CPU 使用率
	if cpu, err := s.getCPUUsage(); err == nil {
		stats.CPUUsage = cpu
	}

	// 4. 內存
	if usage, total, used, err := s.getMemoryUsage(); err == nil {
		stats.MemoryUsage = usage
		stats.MemoryTotal = total
		stats.MemoryUsed = used
	}

	// 5. 磁盤
	if usage, total, used, err := s.getDiskUsage(); err == nil {
		stats.DiskUsage = usage
		stats.DiskTotal = total
		stats.DiskUsed = used
	}

	// 6. 網絡 (速度與總量)
	if tx, rx, err := s.getNetworkSpeed(); err == nil {
		stats.NetworkTX = tx
		stats.NetworkRX = rx
	}
	stats.NetSentTotal = s.lastNetworkIO.TxBytes
	stats.NetRecvTotal = s.lastNetworkIO.RxBytes

	// 7. BBR
	stats.BBR, _ = s.getBBRStatus()

	// 8. IP 地址 (從緩存讀取，不發起網絡請求)
	s.ipMutex.RLock()
	stats.IPv4 = s.publicIPv4
	stats.IPv6 = s.publicIPv6
	s.ipMutex.RUnlock()

	// 如果還沒獲取到，顯示默認值 (View 層會處理空字符串顯示為 "檢查中...")
	if stats.IPv4 == "" {
		// 可選：如果你希望暫時顯示內網 IP 直到公網 IP 加載出來，可以在這裡調用 getIPv4FromCmd
		// 但為了避免混淆，建議讓它保持空，直到 HTTP 請求完成
	}

	return stats, nil
}

// --- 內部實現方法 ---

func (s *SystemInfo) getCPUModel() string {
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "model name") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}
	return "Unknown CPU"
}

func (s *SystemInfo) getLoadAvg() string {
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 3 {
			return strings.Join(fields[:3], " ")
		}
	}
	return "0.00 0.00 0.00"
}

func (s *SystemInfo) getCPUUsage() (float64, error) {
	cur, err := s.readCPUTime()
	if err != nil {
		return 0, err
	}
	prev := s.lastCPUTime
	s.lastCPUTime = cur

	prevTotal := prev.User + prev.Nice + prev.System + prev.Idle + prev.IOWait + prev.Irq + prev.SoftIrq + prev.Steal
	curTotal := cur.User + cur.Nice + cur.System + cur.Idle + cur.IOWait + cur.Irq + cur.SoftIrq + cur.Steal

	prevIdle := prev.Idle + prev.IOWait
	curIdle := cur.Idle + cur.IOWait

	totalDiff := float64(curTotal - prevTotal)
	idleDiff := float64(curIdle - prevIdle)

	if totalDiff == 0 {
		return 0, nil
	}
	usage := (totalDiff - idleDiff) / totalDiff * 100.0
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}
	return usage, nil
}

func (s *SystemInfo) readCPUTime() (CPUTime, error) {
	var c CPUTime
	f, err := os.Open("/proc/stat")
	if err != nil {
		return c, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 5 && fields[0] == "cpu" {
			c.User, _ = strconv.ParseUint(fields[1], 10, 64)
			c.Nice, _ = strconv.ParseUint(fields[2], 10, 64)
			c.System, _ = strconv.ParseUint(fields[3], 10, 64)
			c.Idle, _ = strconv.ParseUint(fields[4], 10, 64)
			if len(fields) >= 6 {
				c.IOWait, _ = strconv.ParseUint(fields[5], 10, 64)
			}
			if len(fields) >= 7 {
				c.Irq, _ = strconv.ParseUint(fields[6], 10, 64)
			}
			if len(fields) >= 8 {
				c.SoftIrq, _ = strconv.ParseUint(fields[7], 10, 64)
			}
			if len(fields) >= 9 {
				c.Steal, _ = strconv.ParseUint(fields[8], 10, 64)
			}
		}
	}
	return c, nil
}

func (s *SystemInfo) getMemoryUsage() (float64, uint64, uint64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, err
	}
	defer f.Close()

	var total, available uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.ParseUint(fields[1], 10, 64)

		if strings.HasPrefix(fields[0], "MemTotal") {
			total = val * 1024
		} else if strings.HasPrefix(fields[0], "MemAvailable") {
			available = val * 1024
		}
	}
	if total == 0 {
		return 0, 0, 0, fmt.Errorf("no mem")
	}
	used := total - available
	return float64(used) / float64(total) * 100.0, total, used, nil
}

func (s *SystemInfo) getDiskUsage() (float64, uint64, uint64, error) {
	out, err := exec.Command("df", "/", "-B1").Output()
	if err != nil {
		return 0, 0, 0, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return 0, 0, 0, fmt.Errorf("df error")
	}
	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 4 {
		return 0, 0, 0, fmt.Errorf("df parse error")
	}
	total, _ := strconv.ParseUint(fields[1], 10, 64)
	used, _ := strconv.ParseUint(fields[2], 10, 64)
	if total == 0 {
		return 0, 0, 0, nil
	}
	return float64(used) / float64(total) * 100.0, total, used, nil
}

func (s *SystemInfo) getNetworkSpeed() (float64, float64, error) {
	cur, err := s.readNetworkIO()
	if err != nil {
		return 0, 0, err
	}

	if s.lastNetworkIO.Time.IsZero() {
		s.lastNetworkIO = cur
		return 0, 0, nil
	}

	diff := cur.Time.Sub(s.lastNetworkIO.Time).Seconds()
	if diff <= 0 {
		return 0, 0, nil
	}

	var tx, rx float64
	if cur.TxBytes >= s.lastNetworkIO.TxBytes {
		tx = float64(cur.TxBytes - s.lastNetworkIO.TxBytes)
	}
	if cur.RxBytes >= s.lastNetworkIO.RxBytes {
		rx = float64(cur.RxBytes - s.lastNetworkIO.RxBytes)
	}

	s.lastNetworkIO = cur
	return tx / diff / 1024 / 1024, rx / diff / 1024 / 1024, nil
}

func (s *SystemInfo) readNetworkIO() (NetworkIO, error) {
	var n NetworkIO
	n.Time = time.Now()
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return n, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) < 2 {
				continue
			}
			iface := strings.TrimSpace(parts[0])
			if iface == "lo" || strings.HasPrefix(iface, "tun") || strings.HasPrefix(iface, "docker") {
				continue
			}

			fields := strings.Fields(parts[1])
			if len(fields) < 9 {
				continue
			}
			rx, _ := strconv.ParseUint(fields[0], 10, 64)
			tx, _ := strconv.ParseUint(fields[8], 10, 64)
			n.RxBytes += rx
			n.TxBytes += tx
		}
	}
	return n, nil
}

func (s *SystemInfo) getBBRStatus() (string, error) {
	data, err := os.ReadFile("/proc/sys/net/ipv4/tcp_congestion_control")
	if err != nil {
		return "unknown", nil
	}
	return strings.TrimSpace(string(data)), nil
}

// 保留此方法作為後備方案
func (s *SystemInfo) getIPv4FromCmd() string {
	out, _ := exec.Command("ip", "-4", "addr", "show", "scope", "global").Output()
	return parseIP(string(out))
}

// 保留此方法作為後備方案
func (s *SystemInfo) getIPv6FromCmd() string {
	out, _ := exec.Command("ip", "-6", "addr", "show", "scope", "global").Output()
	return parseIP(string(out))
}

func parseIP(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "inet") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return strings.Split(fields[1], "/")[0]
			}
		}
	}
	return ""
}

// GetServiceStats 獲取服務狀態
func (s *SystemInfo) GetServiceStats(name string) (*ServiceStats, error) {
	stats := &ServiceStats{
		Status: "not-running",
	}

	cmd := exec.Command("systemctl", "show", name, "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		return stats, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := parts[0], parts[1]

		switch key {
		case "ActiveState":
			stats.Status = val
		case "MainPID":
			pid, _ := strconv.Atoi(val)
			if pid > 0 {
				stats.PID = pid
			}
		case "MemoryCurrent":
			if val != "[not set]" {
				if mem, err := strconv.ParseUint(val, 10, 64); err == nil {
					stats.MemoryBytes = mem
					stats.Memory = formatBytes(mem)
				}
			}
		case "ActiveEnterTimestamp":
			if val != "" {
				layouts := []string{
					"Mon 2006-01-02 15:04:05 MST",
					"2006-01-02 15:04:05 MST",
				}
				for _, l := range layouts {
					if t, err := time.Parse(l, val); err == nil {
						stats.UptimeDur = time.Since(t)
						stats.Uptime = formatDuration(stats.UptimeDur)
						break
					}
				}
			}
		}
	}
	return stats, nil
}

// --- 格式化工具 ---

func formatBytes(bytes uint64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	fBytes := float64(bytes) / 1024.0
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	unitIdx := 0
	for fBytes >= 1024 && unitIdx < len(units)-1 {
		fBytes /= 1024
		unitIdx++
	}
	return fmt.Sprintf("%.2f %s", fBytes, units[unitIdx])
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d秒", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d分鐘", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%d小時%d分鐘", hours, minutes)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%d天%d小時", days, hours)
}

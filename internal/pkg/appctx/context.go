package appctx

import (
	"fmt"
	"os"
	"path/filepath"
)

// Paths 定義應用程序所有的關鍵路徑
type Paths struct {
	BaseDir   string
	ConfigDir string
	DataDir   string
	LogDir    string
	CertDir   string
	BackupDir string

	ConfigFile         string
	CoreBinPath        string
	SystemdServicePath string
}

func NewPaths(baseDir string) (*Paths, error) {
	if baseDir == "" {
		if isProduction() {
			baseDir = "/etc/prism"
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("無法獲取用戶主目錄: %w", err)
			}
			baseDir = filepath.Join(home, ".prism")
		}
	}

	absPath, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("無法解析絕對路徑: %w", err)
	}

	configDir := absPath
	dataDir := filepath.Join(absPath, "data")
	certDir := filepath.Join(absPath, "certs")
	backupDir := filepath.Join(absPath, "backups")
	configFile := filepath.Join(configDir, "config.yaml")

	// 日誌目錄邏輯
	logDir := filepath.Join(absPath, "logs")
	if isProduction() {
		logDir = "/var/log/prism"
	}

	// 默認系統路徑
	coreBinPath := "/usr/local/bin/sing-box"
	servicePath := "/etc/systemd/system/sing-box.service"

	// 開發環境路徑覆蓋
	if !isProduction() {
		coreBinPath = filepath.Join(absPath, "bin", "sing-box")
	}

	paths := &Paths{
		BaseDir:            absPath,
		ConfigDir:          configDir,
		DataDir:            dataDir,
		LogDir:             logDir,
		CertDir:            certDir,
		BackupDir:          backupDir,
		ConfigFile:         configFile,
		CoreBinPath:        coreBinPath,
		SystemdServicePath: servicePath,
	}

	// 確保目錄存在
	dirs := []string{
		paths.ConfigDir,
		paths.DataDir,
		paths.LogDir,
		paths.CertDir,
		paths.BackupDir,
	}

	if !isProduction() {
		dirs = append(dirs, filepath.Dir(paths.CoreBinPath))
	}

	for _, dir := range dirs {
		perm := os.FileMode(0700)
		if dir == paths.LogDir {
			perm = 0755
		}
		if err := os.MkdirAll(dir, perm); err != nil {
			return nil, fmt.Errorf("無法創建目錄 %s: %w", dir, err)
		}
	}

	return paths, nil
}

func isProduction() bool {
	return os.Geteuid() == 0 || os.Getenv("PRISM_ENV") == "production"
}

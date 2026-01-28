package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Yat-Muk/prism-v2/internal/domain/validator"
	"github.com/Yat-Muk/prism-v2/internal/pkg/crypto"
)

const (
	BackupFileMode os.FileMode = 0600
	BackupDirMode  os.FileMode = 0700
	ChecksumSuffix             = ".sha256"
)

type Manager struct {
	backupDir string
	retention RetentionPolicy
	encryptor *crypto.Encryptor
}

type RetentionPolicy struct {
	MaxFiles int
	MaxAge   time.Duration
}

type BackupFile struct {
	Name      string
	Path      string
	ModTime   time.Time
	Size      int64
	Encrypted bool
	Verified  bool
}

// NewManager 修正：這裡必須接收 keyPath string，而不是 []KeyEntry
func NewManager(backupDir string, keyPath string, retention RetentionPolicy) (*Manager, error) {
	// 使用單密鑰初始化
	encryptor, err := crypto.NewEncryptor(keyPath)
	if err != nil {
		return nil, fmt.Errorf("加密器初始化失敗: %w", err)
	}

	if err := os.MkdirAll(backupDir, BackupDirMode); err != nil {
		return nil, fmt.Errorf("創建備份目錄失敗: %w", err)
	}

	return &Manager{
		backupDir: backupDir,
		retention: retention,
		encryptor: encryptor,
	}, nil
}

func (m *Manager) Backup(srcPath string, tag string) error {
	if m.encryptor == nil {
		return fmt.Errorf("加密器未初始化")
	}

	if err := validator.ValidateSafePath(filepath.Dir(srcPath), filepath.Base(srcPath)); err != nil {
		return fmt.Errorf("源文件路徑不安全: %w", err)
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("讀取源文件失敗: %w", err)
	}

	// 簡單去重
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])
	if m.isDuplicateContent(hashStr) {
		return nil
	}

	// 加密
	encryptedStr, err := m.encryptor.Encrypt(string(data))
	if err != nil {
		return fmt.Errorf("加密失敗: %w", err)
	}
	encryptedData := []byte(encryptedStr)

	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("config-%s.bak", timestamp)
	if tag != "" {
		backupName = fmt.Sprintf("config-%s-%s.bak", timestamp, tag)
	}
	dstPath := filepath.Join(m.backupDir, backupName)

	if err := os.WriteFile(dstPath, encryptedData, BackupFileMode); err != nil {
		return fmt.Errorf("寫入備份失敗: %w", err)
	}

	if err := m.saveChecksum(dstPath, encryptedData); err != nil {
		os.Remove(dstPath)
		return fmt.Errorf("生成校驗文件失敗: %w", err)
	}

	m.saveLastHash(hashStr)
	m.enforcePolicy()

	return nil
}

func (m *Manager) saveChecksum(filePath string, data []byte) error {
	checksum := m.encryptor.ComputeHMAC(data)
	tmpPath := filePath + ChecksumSuffix + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(checksum), BackupFileMode); err != nil {
		return err
	}
	return os.Rename(tmpPath, filePath+ChecksumSuffix)
}

func (m *Manager) verifyChecksum(filePath string, data []byte) bool {
	checksumPath := filePath + ChecksumSuffix
	expected, err := os.ReadFile(checksumPath)
	if err != nil {
		return false
	}
	return m.encryptor.VerifyHMAC(data, string(expected))
}

func (m *Manager) Restore(backupName string, targetPath string) error {
	srcPath := filepath.Join(m.backupDir, backupName)
	encryptedData, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("讀取備份文件失敗: %w", err)
	}

	if !m.verifyChecksum(srcPath, encryptedData) {
		return fmt.Errorf("備份完整性校驗失敗")
	}

	decryptedStr, err := m.encryptor.Decrypt(string(encryptedData))
	if err != nil {
		return fmt.Errorf("解密失敗: %w", err)
	}

	tmpFile := targetPath + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(decryptedStr), 0600); err != nil {
		return fmt.Errorf("寫入臨時文件失敗: %w", err)
	}

	if err := os.Rename(tmpFile, targetPath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("替換目標文件失敗: %w", err)
	}

	return nil
}

func (m *Manager) List() ([]BackupFile, error) {
	entries, err := os.ReadDir(m.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupFile{}, nil
		}
		return nil, fmt.Errorf("讀取備份目錄失敗: %w", err)
	}

	var backups []BackupFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".bak") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		path := filepath.Join(m.backupDir, entry.Name())

		encrypted := false
		if f, err := os.Open(path); err == nil {
			buf := make([]byte, 16)
			if n, _ := f.Read(buf); n > 0 {
				encrypted = crypto.IsEncrypted(string(buf[:n]))
			}
			f.Close()
		}

		var verified bool
		if data, err := os.ReadFile(path); err == nil {
			verified = m.verifyChecksum(path, data)
		}

		backups = append(backups, BackupFile{
			Name:      entry.Name(),
			Path:      path,
			ModTime:   info.ModTime(),
			Size:      info.Size(),
			Encrypted: encrypted,
			Verified:  verified,
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})

	return backups, nil
}

func (m *Manager) enforcePolicy() {
	backups, err := m.List()
	if err != nil {
		return
	}

	now := time.Now()
	for i, b := range backups {
		shouldDelete := false
		if m.retention.MaxFiles > 0 && i >= m.retention.MaxFiles {
			shouldDelete = true
		} else if m.retention.MaxAge > 0 && now.Sub(b.ModTime) > m.retention.MaxAge {
			shouldDelete = true
		}

		if shouldDelete {
			os.Remove(b.Path)
			os.Remove(b.Path + ChecksumSuffix)
		}
	}
}

func (m *Manager) isDuplicateContent(hash string) bool {
	hashFile := filepath.Join(m.backupDir, ".last-hash")
	lastHash, err := os.ReadFile(hashFile)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(lastHash)) == hash
}

func (m *Manager) saveLastHash(hash string) {
	hashFile := filepath.Join(m.backupDir, ".last-hash")
	tmpFile := hashFile + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(hash), BackupFileMode); err == nil {
		os.Rename(tmpFile, hashFile)
	}
}

package singbox

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Yat-Muk/prism-v2/internal/pkg/appctx"
	"go.uber.org/zap"
)

type Installer struct {
	log   *zap.Logger
	paths *appctx.Paths
}

func NewInstaller(log *zap.Logger, paths *appctx.Paths) *Installer {
	return &Installer{
		log:   log,
		paths: paths,
	}
}

// InstallLatest 從 GitHub 下載並安裝 Sing-box
func (i *Installer) InstallLatest(version string) error {
	// 1. 環境檢查
	if runtime.GOOS != "linux" {
		return fmt.Errorf("不支持的操作系統: %s", runtime.GOOS)
	}
	arch := runtime.GOARCH
	if arch != "amd64" && arch != "arm64" {
		return fmt.Errorf("不支持的架構: %s", arch)
	}

	// 2. 構建下載 URL
	version = strings.TrimPrefix(version, "v")
	fileName := fmt.Sprintf("sing-box-%s-linux-%s.tar.gz", version, arch)
	url := fmt.Sprintf("https://github.com/SagerNet/sing-box/releases/download/v%s/%s", version, fileName)

	i.log.Info("開始下載核心", zap.String("url", url))

	// 3. 創建帶超時的請求
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("下載請求失敗: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下載失敗，HTTP 狀態碼: %d", resp.StatusCode)
	}

	// 4. 解壓並安裝
	return i.extractAndInstall(resp.Body)
}

func (i *Installer) extractAndInstall(r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("創建 gzip reader 失敗: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	targetPath := i.paths.CoreBinPath
	targetDir := filepath.Dir(targetPath)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("讀取 tar 失敗: %w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		// 尋找二進制文件
		if header.Name == "sing-box" || strings.HasSuffix(header.Name, "/sing-box") {
			i.log.Info("正在安裝核心文件...")

			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return err
			}

			// 原子寫入：寫入臨時文件 -> Rename
			tmpPath := targetPath + ".tmp"
			f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return fmt.Errorf("創建臨時文件失敗: %w", err)
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				os.Remove(tmpPath)
				return fmt.Errorf("寫入文件失敗: %w", err)
			}
			f.Close()

			// 替換舊文件
			if err := os.Rename(tmpPath, targetPath); err != nil {
				return fmt.Errorf("替換核心文件失敗: %w", err)
			}

			i.log.Info("核心安裝成功", zap.String("path", targetPath))
			return nil
		}
	}

	return fmt.Errorf("在壓縮包中未找到 sing-box 二進制文件")
}

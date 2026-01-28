package system

import (
	"fmt"
	"os"
	"os/exec"
	"text/template"
)

const serviceTemplate = `[Unit]
Description=Sing-box service
Documentation=https://sing-box.sagernet.org
After=network.target nss-lookup.target

[Service]
Environment="ENABLE_DEPRECATED_LEGACY_DNS_SERVERS=true"
Environment="ENABLE_DEPRECATED_LEGACY_DOMAIN_STRATEGY_OPTIONS=true"
Environment="ENABLE_DEPRECATED_SPECIAL_OUTBOUNDS=true"
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW
Group=root
User=root
ExecStart={{.BinPath}} run -c {{.ConfigPath}}
Restart=on-failure
RestartSec=3s
LimitNOFILE=infinity

[Install]
WantedBy=multi-user.target
`

type ServiceInstaller struct {
	BinPath    string
	ConfigPath string
}

func NewServiceInstaller(binPath, configPath string) *ServiceInstaller {
	return &ServiceInstaller{
		BinPath:    binPath,
		ConfigPath: configPath,
	}
}

// Install 寫入並啟用 Systemd 服務
func (s *ServiceInstaller) Install(servicePath string) error {

	f, err := os.Create(servicePath)
	if err != nil {
		return fmt.Errorf("無法創建服務文件: %w", err)
	}
	defer f.Close()

	tmpl, err := template.New("service").Parse(serviceTemplate)
	if err != nil {
		return err
	}

	if err := tmpl.Execute(f, s); err != nil {
		return err
	}

	// 重載 Systemd
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("daemon-reload 失敗: %w", err)
	}

	// 啟用服務
	if err := exec.Command("systemctl", "enable", "sing-box").Run(); err != nil {
		return fmt.Errorf("啟用服務失敗: %w", err)
	}

	return nil
}

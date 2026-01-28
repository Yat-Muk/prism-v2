package protocol

import (
	"path/filepath"

	domainConfig "github.com/Yat-Muk/prism-v2/internal/domain/config"
	"github.com/Yat-Muk/prism-v2/internal/pkg/appctx"
)

// Factory 協議工廠接口
type Factory interface {
	FromConfig(cfg *domainConfig.Config) []Protocol
}

// NewFactory 創建協議工廠
func NewFactory(paths *appctx.Paths) Factory {
	return &factoryImpl{
		paths: paths,
	}
}

type factoryImpl struct {
	paths *appctx.Paths
}

// 根據證書配置推導一個統一的 TLS 域名（可選）
func getTLSDomain(p *domainConfig.ProtocolsConfig) string {
	if p.Hysteria2.CertMode == "acme" && p.Hysteria2.CertDomain != "" {
		return p.Hysteria2.CertDomain
	}
	if p.TUIC.CertMode == "acme" && p.TUIC.CertDomain != "" {
		return p.TUIC.CertDomain
	}
	if p.AnyTLS.CertMode == "acme" && p.AnyTLS.CertDomain != "" {
		return p.AnyTLS.CertDomain
	}
	return "www.bing.com"
}

// getSNI 動態獲取 SNI（優先 ACME 域名）
func getSNI(certMode, certDomain, configSNI string, globalTLSDomain string) string {
	// ACME 模式：強制使用 cert_domain
	if certMode == "acme" && certDomain != "" {
		return certDomain
	}

	// 自簽名模式：使用配置的 SNI 或默認域名
	if configSNI != "" {
		return configSNI
	}

	// 最終 fallback
	if globalTLSDomain != "" {
		return globalTLSDomain
	}

	return "www.bing.com"
}

// getCertPath 根據證書模式獲取證書路徑
func (f *factoryImpl) getCertPath(certMode, certDomain string) (certPath, keyPath string) {
	baseDir := f.paths.CertDir

	if certMode == "acme" && certDomain != "" {
		certPath = filepath.Join(baseDir, certDomain+".crt")
		keyPath = filepath.Join(baseDir, certDomain+".key")
	} else {
		certPath = filepath.Join(baseDir, "self_signed.crt")
		keyPath = filepath.Join(baseDir, "self_signed.key")
	}
	return certPath, keyPath
}

// FromConfig 從 YAML 配置創建協議實例
func (f *factoryImpl) FromConfig(cfg *domainConfig.Config) []Protocol {

	var protocols []Protocol

	// 1. Reality Vision
	if cfg.Protocols.RealityVision.Enabled {
		protocols = append(protocols, &RealityVision{
			BaseProtocol: BaseProtocol{
				type_:   TypeRealityVision,
				name:    "Reality Vision",
				port:    cfg.Protocols.RealityVision.Port,
				enabled: true,
			},
			SNI:        cfg.Protocols.RealityVision.SNI,
			PublicKey:  cfg.Protocols.RealityVision.PublicKey,
			PrivateKey: cfg.Protocols.RealityVision.PrivateKey,
			ShortID:    cfg.Protocols.RealityVision.ShortID,
			Users:      []User{{UUID: cfg.UUID, Flow: "xtls-rprx-vision"}},
		})
	}

	// 2. Reality gRPC
	if cfg.Protocols.RealityGRPC.Enabled {
		protocols = append(protocols, &RealityGRPC{
			BaseProtocol: BaseProtocol{
				type_:   TypeRealityGRPC,
				name:    "Reality gRPC",
				port:    cfg.Protocols.RealityGRPC.Port,
				enabled: true,
			},
			SNI:         cfg.Protocols.RealityGRPC.SNI,
			PublicKey:   cfg.Protocols.RealityGRPC.PublicKey,
			PrivateKey:  cfg.Protocols.RealityGRPC.PrivateKey,
			ShortID:     cfg.Protocols.RealityGRPC.ShortID,
			ServiceName: "grpc",
			Users:       []User{{UUID: cfg.UUID, Flow: ""}},
		})
	}

	// 3. Hysteria2
	if cfg.Protocols.Hysteria2.Enabled {
		sni := getSNI(
			cfg.Protocols.Hysteria2.CertMode,
			cfg.Protocols.Hysteria2.CertDomain,
			cfg.Protocols.Hysteria2.SNI,
			getTLSDomain(&cfg.Protocols),
		)

		// 調用方法獲取路徑
		certPath, keyPath := f.getCertPath(
			cfg.Protocols.Hysteria2.CertMode,
			cfg.Protocols.Hysteria2.CertDomain,
		)

		// 從配置讀取帶寬，如果為 0 則使用默認值 100
		upMbps := cfg.Protocols.Hysteria2.UpMbps
		if upMbps == 0 {
			upMbps = 100
		}
		downMbps := cfg.Protocols.Hysteria2.DownMbps
		if downMbps == 0 {
			downMbps = 100
		}

		protocols = append(protocols, &Hysteria2{
			BaseProtocol: BaseProtocol{
				type_:   TypeHysteria2,
				name:    "Hysteria2",
				port:    cfg.Protocols.Hysteria2.Port,
				enabled: true,
			},
			Password:    cfg.Password,
			CertPath:    certPath,
			KeyPath:     keyPath,
			SNI:         sni,
			ALPN:        "h3",
			UpMbps:      upMbps,
			DownMbps:    downMbps,
			Obfs:        cfg.Protocols.Hysteria2.Obfs,
			PortHopping: cfg.Protocols.Hysteria2.PortHopping,
		})
	}

	// 4. TUIC
	if cfg.Protocols.TUIC.Enabled {
		sni := getSNI(
			cfg.Protocols.TUIC.CertMode,
			cfg.Protocols.TUIC.CertDomain,
			cfg.Protocols.TUIC.SNI,
			getTLSDomain(&cfg.Protocols),
		)

		certPath, keyPath := f.getCertPath(
			cfg.Protocols.TUIC.CertMode,
			cfg.Protocols.TUIC.CertDomain,
		)

		protocols = append(protocols, &TUIC{
			BaseProtocol: BaseProtocol{
				type_:   TypeTUIC,
				name:    "TUIC",
				port:    cfg.Protocols.TUIC.Port,
				enabled: true,
			},
			UUID:              cfg.UUID,
			Password:          cfg.Password,
			SNI:               sni,
			CertPath:          certPath,
			KeyPath:           keyPath,
			ALPN:              []string{"h3"},
			CongestionControl: "bbr",
			ZeroRTTHandshake:  false,
		})
	}

	// 5. AnyTLS
	if cfg.Protocols.AnyTLS.Enabled {
		sni := getSNI(
			cfg.Protocols.AnyTLS.CertMode,
			cfg.Protocols.AnyTLS.CertDomain,
			cfg.Protocols.AnyTLS.SNI,
			getTLSDomain(&cfg.Protocols),
		)

		certPath, keyPath := f.getCertPath(
			cfg.Protocols.AnyTLS.CertMode,
			cfg.Protocols.AnyTLS.CertDomain,
		)

		protocols = append(protocols, &AnyTLS{
			BaseProtocol: BaseProtocol{
				type_:   TypeAnyTLS,
				name:    "AnyTLS",
				port:    cfg.Protocols.AnyTLS.Port,
				enabled: true,
			},
			Username:    "prism",
			Password:    cfg.Password,
			SNI:         sni,
			CertPath:    certPath,
			KeyPath:     keyPath,
			PaddingMode: cfg.Protocols.AnyTLS.PaddingMode,
			ALPN:        []string{"h2", "http/1.1"},
		})
	}

	// 6. AnyTLS Reality
	if cfg.Protocols.AnyTLSReality.Enabled {
		protocols = append(protocols, &AnyTLSReality{
			BaseProtocol: BaseProtocol{
				type_:   TypeAnyTLSReality,
				name:    "AnyTLS Reality",
				port:    cfg.Protocols.AnyTLSReality.Port,
				enabled: true,
			},
			Username: "prism",
			Password: cfg.Password,
			SNI:      cfg.Protocols.RealityVision.SNI,

			PublicKey:   cfg.Protocols.RealityVision.PublicKey,
			PrivateKey:  cfg.Protocols.RealityVision.PrivateKey,
			ShortID:     cfg.Protocols.RealityVision.ShortID,
			PaddingMode: cfg.Protocols.AnyTLSReality.PaddingMode,
			ALPN:        []string{"h2", "http/1.1"},
		})
	}

	// 7. ShadowTLS
	if cfg.Protocols.ShadowTLS.Enabled {
		protocols = append(protocols, &ShadowTLS{
			BaseProtocol: BaseProtocol{
				type_:   TypeShadowTLS,
				name:    "ShadowTLS v3",
				port:    cfg.Protocols.ShadowTLS.Port,
				enabled: true,
			},
			Password:   cfg.Password,
			SSPassword: cfg.Password,
			SSMethod:   "2022-blake3-aes-128-gcm",
			SNI:        cfg.Protocols.ShadowTLS.SNI,
			DetourPort: 10000,
			StrictMode: true,
		})
	}

	return protocols
}

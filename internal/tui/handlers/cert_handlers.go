package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Yat-Muk/prism-v2/internal/pkg/inputvalidator"
	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/state"
	tea "github.com/charmbracelet/bubbletea"
)

type CertHandler struct {
	cmdBuilder *CommandBuilder
}

func NewCertHandler(cmdBuilder *CommandBuilder) *CertHandler {
	return &CertHandler{cmdBuilder: cmdBuilder}
}

// Handle 是證書模塊的統一入口
func (h *CertHandler) Handle(msg tea.KeyMsg, m *state.Manager) (*state.Manager, tea.Cmd) {
	switch m.UI().CurrentView {
	case state.CertMenuView:
		return h.handleCertMenu(m)
	case state.ACMEHTTPInputView:
		return h.handleHTTPInput(m)
	case state.ACMEDNSProviderView:
		return h.handleProviderSelect(m)
	case state.DNSCredentialInputView:
		return h.handleCredentialInput(m)
	case state.CertRenewView:
		return h.handleRenew(m)
	case state.CertDeleteView:
		return h.handleDelete(m)
	case state.ProviderSwitchView:
		return h.handleSwitchProvider(m)
	case state.CertStatusView:
		return h.handleStatus(m)
	case state.CertModeMenuView:
		return h.handleModeSwitch(m)
	}

	m.UI().ClearInput()
	return m, nil
}

// handleCertMenu 處理證書主菜單
func (h *CertHandler) handleCertMenu(m *state.Manager) (*state.Manager, tea.Cmd) {
	input := strings.TrimSpace(m.UI().GetInputBuffer())
	m.UI().ClearInput()

	switch input {
	case constants.KeyCert_ApplyHTTP:
		m.Cert().ResetHTTPStep()
		m.UI().SetStatus(state.StatusInfo, "正在檢測 80 端口可用性...", "請稍候", true)
		return m, h.cmdBuilder.CheckPort80Cmd()

	case constants.KeyCert_ApplyDNS:
		m.UI().SwitchView(state.ACMEDNSProviderView)
		m.Cert().SelectedDomain = ""
		m.Cert().SelectedProvider = ""
	case constants.KeyCert_SwitchProvider:
		m.UI().SwitchView(state.ProviderSwitchView)
	case constants.KeyCert_Renew:
		m.UI().SwitchView(state.CertRenewView)
		return m, h.cmdBuilder.RefreshCertListCmd()
	case constants.KeyCert_Status:
		m.UI().SwitchView(state.CertStatusView)
		return m, h.cmdBuilder.RefreshCertListCmd()
	case constants.KeyCert_Delete:
		m.UI().SwitchView(state.CertDeleteView)
		return m, h.cmdBuilder.RefreshCertListCmd()
	case constants.KeyCert_ModeSwitch:
		m.UI().SwitchView(state.CertModeMenuView)
	}
	return m, nil
}

// handleHTTPInput 統一處理 郵箱(Step0) 和 域名(Step1)
func (h *CertHandler) handleHTTPInput(m *state.Manager) (*state.Manager, tea.Cmd) {
	input := strings.TrimSpace(m.UI().GetInputBuffer())
	m.UI().ClearInput()

	step := m.Cert().HTTPStep

	switch step {
	case 0: // --- 步驟 1: 輸入郵箱 ---
		if input == "" {
			// 空輸入 -> 生成隨機郵箱
			randomID := time.Now().Unix() % 10000
			input = fmt.Sprintf("prism_user_%d@gmail.com", randomID)
		} else {
			if err := inputvalidator.ValidateEmail(input); err != nil {
				m.UI().SetStatus(state.StatusError, err.Error(), "請重新輸入有效的郵箱地址", false)
				return m, nil
			}
		}

		// 保存郵箱
		m.Cert().ACMEEmail = input

		// 進入下一步
		m.Cert().NextHTTPStep()

		// 更新 UI 提示
		m.UI().SetStatus(state.StatusInfo, "請輸入域名", "", false)
		return m, nil

	case 1: // --- 步驟 2: 輸入域名 ---
		// 域名不能為空
		if input == "" {
			return m, nil
		}

		// 只有在第二步才校驗域名格式
		if err := inputvalidator.ValidateDomainInput(input); err != nil {
			m.UI().SetStatus(state.StatusError, err.Error(), "", false)
			return m, nil
		}

		m.Cert().SelectedDomain = input
		email := m.Cert().ACMEEmail

		m.UI().SetStatus(state.StatusInfo, "正在申請證書...", "請勿關閉程序 (約 1-2 分鐘)", true)

		// 發起申請
		return m, h.cmdBuilder.RequestACMEHTTPCmd(input, email)
	}

	return m, nil
}

// handleProviderSelect 處理 DNS API 提供商選擇
func (h *CertHandler) handleProviderSelect(m *state.Manager) (*state.Manager, tea.Cmd) {
	input := strings.TrimSpace(m.UI().GetInputBuffer())
	m.UI().ClearInput()

	// 映射輸入到提供商標識
	providers := map[string]string{
		constants.KeyProvider_Cloudflare: "cloudflare",
		constants.KeyProvider_Aliyun:     "alidns",
		constants.KeyProvider_DNSPod:     "dnspod",
		constants.KeyProvider_AWS:        "aws",
		constants.KeyProvider_Google:     "googlecloud",
	}

	if p, ok := providers[input]; ok {
		m.Cert().SelectedProvider = p
		m.Cert().ResetDNSStep()
		m.UI().SwitchView(state.DNSCredentialInputView)
	} else {
		if input != "" {
			m.UI().SetStatus(state.StatusError, "無效的選項", "", false)
		}
	}
	return m, nil
}

// handleCredentialInput 處理 DNS 申請的分步輸入 (域名 -> ID -> Secret)
func (h *CertHandler) handleCredentialInput(m *state.Manager) (*state.Manager, tea.Cmd) {
	input := strings.TrimSpace(m.UI().GetInputBuffer())
	m.UI().ClearInput()

	// [FIX] 直接訪問字段
	currentStep := m.Cert().DNSStep

	switch currentStep {
	case 0: // 第一步：輸入域名
		if input == "" {
			return m, nil
		}
		if err := inputvalidator.ValidateDomainInput(input); err != nil {
			m.UI().SetStatus(state.StatusError, err.Error(), "", false)
			return m, nil
		}
		// [FIX] 直接賦值
		m.Cert().SelectedDomain = input
		m.Cert().NextDNSStep()
		return m, nil

	case 1: // 第二步：輸入 Access Key ID / Token
		if input == "" {
			m.UI().SetStatus(state.StatusError, "ID/Token 不能為空", "", false)
			return m, nil
		}
		// [FIX] 直接賦值
		m.Cert().DNSProviderID = input
		m.Cert().NextDNSStep()
		return m, nil

	case 2: // 第三步：輸入 Secret Key 並提交
		if input == "" {
			m.UI().SetStatus(state.StatusError, "Secret 不能為空", "", false)
			return m, nil
		}
		// [FIX] 直接賦值
		m.Cert().DNSProviderSecret = input

		// [FIX] 直接從 State 收集數據
		domain := m.Cert().SelectedDomain
		provider := m.Cert().SelectedProvider
		id := m.Cert().DNSProviderID
		secret := m.Cert().DNSProviderSecret

		m.Cert().ResetDNSStep()

		// [重要] 保持在當前頁面等待結果
		m.UI().SetStatus(state.StatusInfo, "正在通過 DNS API 申請證書...", "這可能需要 1-2 分鐘等待 DNS 傳播", true)

		return m, h.cmdBuilder.RequestACMEDNSCmd(domain, provider, id, secret)
	}

	return m, nil
}

// handleRenew 處理證書續期
func (h *CertHandler) handleRenew(m *state.Manager) (*state.Manager, tea.Cmd) {
	input := strings.TrimSpace(m.UI().GetInputBuffer())
	m.UI().ClearInput()

	// 選項 1: 續期所有
	if input == constants.KeyRenew_All {
		m.UI().SetStatus(state.StatusInfo, "正在批量續期所有證書...", "請稍候", true)
		return m, h.cmdBuilder.RenewAllCertsCmd()
	}

	// 選項 2+: 單個續期
	if idx, err := strconv.Atoi(input); err == nil {
		// [FIX] 直接訪問 Slice
		list := m.Cert().ACMEDomains

		offset := 2
		realIdx := idx - offset

		if realIdx >= 0 && realIdx < len(list) {
			domain := list[realIdx]
			m.UI().SetStatus(state.StatusInfo, "正在續期: "+domain, "請稍候", true)
			return m, h.cmdBuilder.RenewCertCmd(domain)
		}
	}
	return m, nil
}

// handleDelete 處理證書刪除
func (h *CertHandler) handleDelete(m *state.Manager) (*state.Manager, tea.Cmd) {
	input := strings.TrimSpace(m.UI().GetInputBuffer())
	m.UI().ClearInput()

	// [FIX] IsConfirmMode 改為直接讀取字段
	if m.Cert().CertConfirmMode {
		if strings.EqualFold(input, "YES") {
			// [FIX] 直接讀取字段
			domain := m.Cert().CertToDelete
			m.Cert().ResetCertDeletion()
			m.UI().SetStatus(state.StatusInfo, "正在刪除證書...", domain, true)
			return m, h.cmdBuilder.DeleteCertCmd(domain)
		}
		m.Cert().ResetCertDeletion()
		m.UI().SetStatus(state.StatusInfo, "已取消刪除", "", false)
		return m, nil
	}

	// 選擇要刪除的證書
	if idx, err := strconv.Atoi(input); err == nil {
		// [FIX] 直接訪問 Slice
		list := m.Cert().ACMEDomains

		if idx >= 1 && idx <= len(list) {
			target := list[idx-1]
			m.Cert().PrepareCertDeletion(target)
			m.UI().SetStatus(state.StatusWarn, fmt.Sprintf("確認刪除 %s? 輸入 YES", target), "", true)
			return m, nil
		}
	}
	return m, nil
}

// handleSwitchProvider 處理 CA 提供商切換 (Let's Encrypt / ZeroSSL)
func (h *CertHandler) handleSwitchProvider(m *state.Manager) (*state.Manager, tea.Cmd) {
	input := strings.TrimSpace(m.UI().GetInputBuffer())
	m.UI().ClearInput()

	var p string
	switch input {
	case constants.KeyProviderType_LetsEncrypt:
		p = "letsencrypt"
	case constants.KeyProviderType_ZeroSSL:
		p = "zerossl"
	}

	if p != "" {
		m.UI().SetStatus(state.StatusInfo, "正在切換 CA 賬戶...", "可能需要註冊 EAB，請查看日誌", true)
		m.UI().SwitchView(state.CertMenuView)
		return m, h.cmdBuilder.SwitchProviderCmd(p)
	}
	return m, nil
}

// handleStatus 處理狀態查看頁面
func (h *CertHandler) handleStatus(m *state.Manager) (*state.Manager, tea.Cmd) {
	m.UI().ClearInput()
	return m, nil
}

// handleModeSwitch 處理模式切換
func (h *CertHandler) handleModeSwitch(m *state.Manager) (*state.Manager, tea.Cmd) {
	input := strings.TrimSpace(m.UI().GetInputBuffer())
	m.UI().ClearInput()

	var protocol string
	var label string

	switch input {
	case "1":
		protocol = "hysteria2"
		label = "Hysteria 2"
	case "2":
		protocol = "tuic"
		label = "TUIC v5"
	case "3":
		protocol = "anytls"
		label = "AnyTLS"
	default:
		return m, nil
	}

	m.UI().SetStatus(state.StatusInfo, fmt.Sprintf("正在切換 %s 證書模式...", label), "請稍候", true)

	return m, h.cmdBuilder.ToggleCertModeCmd(protocol)
}

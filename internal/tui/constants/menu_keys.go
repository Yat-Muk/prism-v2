package constants

const (
	// ==========================================
	// 主菜單 (Main Menu)
	// ==========================================
	KeyMain_InstallWizard = "1"  // 安裝部署/重新部署
	KeyMain_ServiceStart  = "2"  // 啟動/重啟服務
	KeyMain_ServiceStop   = "3"  // 停止服務
	KeyMain_Config        = "4"  // 配置與協議
	KeyMain_Cert          = "5"  // 證書管理
	KeyMain_Outbound      = "6"  // 出口策略
	KeyMain_Route         = "7"  // 路由與分流
	KeyMain_Core          = "8"  // 核心與更新
	KeyMain_Tools         = "9"  // 實用工具箱
	KeyMain_Log           = "10" // 實時日誌
	KeyMain_NodeInfo      = "11" // 節點信息
	KeyMain_Uninstall     = "u"  // 卸載 Prism
	KeyMain_Quit          = "q"  // 退出程序

	// ==========================================
	// 配置菜單 (Config Menu)
	// ==========================================
	KeyConfig_Protocol = "1" // 協議開關管理
	KeyConfig_SNI      = "2" // 修改 SNI 域名
	KeyConfig_UUID     = "3" // 修改 UUID
	KeyConfig_Port     = "4" // 修改監聽端口
	KeyConfig_Padding  = "5" // AnyTLS 填充策略
	KeyConfig_Apply    = "s" // 應用配置
	KeyConfig_Reset    = "r" // 重置配置

	// AnyTLS Padding 策略
	KeyPadding_Balanced   = "1" // 均衡流
	KeyPadding_Minimal    = "2" // 極簡流
	KeyPadding_HighResist = "3" // 高對抗流
	KeyPadding_Video      = "4" // 視頻特徵
	KeyPadding_Official   = "5" // 官方默認

	// ==========================================
	// 證書菜單 (Cert Menu)
	// ==========================================
	KeyCert_ApplyHTTP      = "1" // 申請 ACME 證書 (HTTP)
	KeyCert_ApplyDNS       = "2" // 申請 ACME 證書 (DNS)
	KeyCert_SwitchProvider = "3" // 切換證書提供商
	KeyCert_Renew          = "4" // 續期現有證書
	KeyCert_Status         = "5" // 查看證書狀態
	KeyCert_Delete         = "6" // 刪除域名證書
	KeyCert_ModeSwitch     = "7" // 切換證書模式

	// ACME 提供商
	KeyProvider_Cloudflare = "1" // Cloudflare
	KeyProvider_Aliyun     = "2" // 阿里雲 DNS
	KeyProvider_DNSPod     = "3" // 騰訊雲 DNSPod
	KeyProvider_AWS        = "4" // AWS Route53
	KeyProvider_Google     = "5" // Google Cloud DNS

	// ACME 提供商類型
	KeyProviderType_LetsEncrypt = "1" // Let's Encrypt
	KeyProviderType_ZeroSSL     = "2" // ZeroSSL

	// 證書續期
	KeyRenew_All = "1" // 續期所有證書

	// ==========================================
	// 路由菜單 (Route Menu)
	// ==========================================
	KeyRoute_WARP     = "1" // WARP 分流管理
	KeyRoute_Socks5   = "2" // Socks5 分流配置
	KeyRoute_IPv6     = "3" // IPv6 分流配置
	KeyRoute_DNS      = "4" // DNS 分流配置
	KeyRoute_SNIProxy = "5" // SNI 反向代理

	// ==========================================
	// 核心菜單 (Core Menu)
	// ==========================================
	// 已安裝狀態
	KeyCore_CheckUpdate   = "1" // 檢查更新
	KeyCore_Update        = "2" // 更新核心
	KeyCore_Reinstall     = "3" // 重新安裝
	KeyCore_SelectVersion = "4" // 安裝指定版本
	KeyCore_SelectSource  = "5" // 切換更新源
	KeyCore_Uninstall     = "6" // 卸載核心
	KeyScript_Update      = "7" // 腳本更新

	// 腳本更新
	KeyScriptUpdate_Confirm = "1" // 更新腳本
	KeyScriptUpdate_Cancel  = "2" // 取消更新

	// 未安裝狀態
	KeyCore_InstallLatest = "1" // 安裝最新版
	KeyCore_InstallDev    = "3" // 安裝開發版

	// 核心源選擇
	KeySource_Github  = "1" // GitHub
	KeySource_GhProxy = "2" // GhProxy 鏡像

	// ==========================================
	// 服務菜單 (Service Menu)
	// ==========================================
	KeyService_Restart   = "1" // 重啟服務
	KeyService_Stop      = "2" // 停止服務
	KeyService_Log       = "3" // 查看日誌
	KeyService_Refresh   = "4" // 刷新狀態
	KeyService_AutoStart = "5" // 開機自啟
	KeyService_Health    = "6" // 健康檢查

	// ==========================================
	// 工具菜單 (Tools Menu)
	// ==========================================
	KeyTools_Streaming = "1" // 流媒體/IP 檢測
	KeyTools_Swap      = "2" // 虛擬內存 (Swap)
	KeyTools_Fail2Ban  = "3" // Fail2Ban 防護
	KeyTools_TimeSync  = "4" // 校准服務器時間
	KeyTools_BBR       = "5" // BBR 加速與優化
	KeyTools_Cleanup   = "6" // 系統清理
	KeyTools_Backup    = "7" // 配置備份

	// Swap 子菜單
	KeySwap_Create = "1" // 創建 Swap
	KeySwap_Custom = "2" // 自定義大小
	KeySwap_Delete = "3" // 刪除 Swap
	KeySwap_Status = "4" // 查看狀態

	// Fail2Ban 子菜單
	KeyFail2Ban_Install   = "1" // 安裝 Fail2Ban
	KeyFail2Ban_Toggle    = "2" // 啟動/停止
	KeyFail2Ban_List      = "3" // 查看封禁 IP
	KeyFail2Ban_Unban     = "4" // 解封 IP
	KeyFail2Ban_Config    = "5" // 配置規則
	KeyFail2Ban_Uninstall = "6" // 卸載 Fail2Ban

	// 清理子菜單
	KeyCleanup_Scan = "1" // 掃描可清理空間
	KeyCleanup_Log  = "2" // 清理系統日誌
	KeyCleanup_Pkg  = "3" // 清理軟件包緩存
	KeyCleanup_Temp = "4" // 清理臨時文件
	KeyCleanup_All  = "5" // 一鍵清理所有

	// 備份子菜單
	KeyBackup_Create  = "1" // 備份當前配置
	KeyBackup_Restore = "2" // 一鍵恢復備份
	KeyBackup_Delete  = "3" // 刪除指定備份

	// 流媒體檢測子菜單
	KeyStreaming_Run    = "1" // 開始檢測
	KeyStreaming_IP     = "2" // 僅檢測 IP
	KeyStreaming_Report = "3" // 詳細報告

	// BBR 子菜單
	KeyBBR_Original = "1" // 啓用原版 BBR
	KeyBBR_BBR2     = "2" // 啓用 BBR2
	KeyBBR_XanMod   = "3" // 安裝 XanMod 內核
	KeyBBR_Disable  = "4" // 禁用 BBR

	// ==========================================
	// 日誌菜單 (Log Menu)
	// ==========================================
	KeyLog_Realtime = "1" // 查看實時日誌
	KeyLog_Full     = "2" // 查看完整日誌
	KeyLog_Error    = "3" // 查看錯誤日誌
	KeyLog_Level    = "4" // 修改日誌級別
	KeyLog_Export   = "5" // 導出日誌
	KeyLog_Clear    = "6" // 清空日誌

	// 日誌級別
	KeyLevel_Debug = "1" // Debug
	KeyLevel_Info  = "2" // Info
	KeyLevel_Warn  = "3" // Warn
	KeyLevel_Error = "4" // Error

	// ==========================================
	// 節點信息 (Node Info)
	// ==========================================
	KeyNode_Links        = "1" // 查看協議鏈接
	KeyNode_QRCode       = "2" // 生成二維碼
	KeyNode_Subscription = "3" // 查看訂閱鏈接
	KeyNode_ClientConfig = "4" // 導出客戶端配置
	KeyNode_Copy         = "a" // 複製所有鏈接

	// 訂閱子菜單
	KeySubscription_CopyOnline  = "1" // 複製在線訂閱
	KeySubscription_CopyOffline = "2" // 複製離線訂閱
	KeySubscription_Refresh     = "3" // 刷新訂閱
	KeySubscription_QRCode      = "4" // 生成訂閱二維碼

	// 客戶端配置導出 (Client Config)
	KeyExport_Full   = "1" // 導出完整配置
	KeyExport_Clash  = "2" // 導出 Clash 配置
	KeyExport_Custom = "3" // 節點參數

	// ==========================================
	// 端口編輯 (Port Edit)
	// ==========================================
	KeyPort_Reset        = "r" // 重置端口
	KeyPort_Main         = "1" // Hy2 主端口
	KeyPort_Hopping      = "2" // Hy2 端口跳躍
	KeyPort_ClearHopping = "3" // Hy2 清除跳躍

	// ==========================================
	// UUID 編輯
	// ==========================================
	KeyUUID_Generate = "1" // 自動生成 UUID
	KeyUUID_Manual   = "2" // 手動輸入 UUID

	// ==========================================
	// 出站策略 (Outbound Strategy)
	// ==========================================
	KeyOutbound_PreferIPv4 = "1" // IPv4 優先
	KeyOutbound_PreferIPv6 = "2" // IPv6 優先
	KeyOutbound_IPv4Only   = "3" // 僅 IPv4
	KeyOutbound_IPv6Only   = "4" // 僅 IPv6

	// ==========================================
	// WARP 分流 (WARP Routing)
	// ==========================================
	KeyWARP_ToggleIPv4 = "1" // 啟用 WARP IPv4
	KeyWARP_ToggleIPv6 = "2" // 啟用 WARP IPv6
	KeyWARP_SetGlobal  = "3" // 設置全局模式
	KeyWARP_SetDomains = "4" // 添加分流域名
	KeyWARP_ShowConfig = "5" // 查看配置
	KeyWARP_Disable    = "6" // 禁用 WARP
	KeyWARP_SetLicense = "7" // 配置密鑰

	// ==========================================
	// WARP 出站管理 (WARP Config)
	// ==========================================
	KeyWARPConfig_Enable  = "1" // 啟用 WARP
	KeyWARPConfig_Disable = "2" // 禁用 WARP
	KeyWARPConfig_License = "3" // 配置許可證密鑰
	KeyWARPConfig_Test    = "4" // 測試連接

	// ==========================================
	// Socks5 管理
	// ==========================================
	KeySocks5_Inbound    = "1" // Socks5 入站
	KeySocks5_Outbound   = "2" // Socks5 出站
	KeySocks5_ShowConfig = "3" // 查看配置
	KeySocks5_Uninstall  = "4" // 卸載

	// Socks5 入站
	KeySocks5In_Toggle = "1" // 啟動/停止
	KeySocks5In_Port   = "2" // 設置端口
	KeySocks5In_Auth   = "3" // 設置認證
	KeySocks5In_IP     = "4" // 設置允許IP
	KeySocks5In_Rule   = "5" // 分流規則

	// Socks5 出站
	KeySocks5Out_Toggle = "1" // 啟動/停止
	KeySocks5Out_Server = "2" // 設置落地機
	KeySocks5Out_Auth   = "3" // 設置認證
	KeySocks5Out_Global = "4" // 全局轉發
	KeySocks5Out_Rule   = "5" // 分流規則

	// ==========================================
	// IPv6 管理
	// ==========================================
	// IPv6 Route (一般路由設置)
	KeyIPv6Route_Enable   = "1" // 啟用 IPv6 路由
	KeyIPv6Route_Disable  = "2" // 禁用 IPv6 路由
	KeyIPv6Route_Priority = "3" // IPv6 優先級
	KeyIPv6Route_DNS      = "4" // IPv6 DNS

	// IPv6 Routing (分流設置)
	KeyIPv6Split_Enable    = "1" // 啟用 IPv6 分流
	KeyIPv6Split_Disable   = "2" // 禁用 IPv6 分流
	KeyIPv6Split_SetGlobal = "3" // 設置全局 IPv6
	KeyIPv6Split_SetDomain = "4" // 添加分流域名

	// ==========================================
	// DNS & SNI Proxy
	// ==========================================
	// SNI Proxy
	KeySNIProxy_Enable  = "1" // 啟用 SNI 代理
	KeySNIProxy_Disable = "2" // 禁用 SNI 代理
	KeySNIProxy_Add     = "3" // 添加分流規則
	KeySNIProxy_List    = "4" // 查看規則列表
	KeySNIProxy_Delete  = "5" // 刪除規則

	// DNS Routing
	KeyRouting_Enable    = "1" // 啟用 DNS 分流
	KeyRouting_Disable   = "2" // 禁用 DNS 分流
	KeyRouting_AddDomain = "3" // 添加分流域名
	KeyRouting_Show      = "4" // 查看配置
	KeyRouting_DelRule   = "5" // 刪除規則 (新增)

	// DNS Config
	KeyDNS_Toggle   = "1" // 啟用/禁用 DNS
	KeyDNS_Servers  = "2" // 修改 DNS 服務器
	KeyDNS_Strategy = "3" // 修改 DNS 策略
	KeyDNS_Rules    = "4" // 配置 DNS 規則

	// ==========================================
	// 卸載選項 (Uninstall)
	// ==========================================
	KeyUninstall_KeepConfig  = "1"   // 保留配置文件
	KeyUninstall_KeepCert    = "2"   // 保留證書文件
	KeyUninstall_KeepBackup  = "3"   // 保留備份文件
	KeyUninstall_KeepLog     = "4"   // 保留日誌文件
	KeyUninstall_ConfirmStep = "c"   // 繼續卸載
	KeyUninstall_ConfirmYes  = "yes" // 確認卸載
)

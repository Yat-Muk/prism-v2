package view

import (
	"fmt"
	"strings"

	"github.com/Yat-Muk/prism-v2/internal/tui/constants"
	"github.com/Yat-Muk/prism-v2/internal/tui/style"
	"github.com/Yat-Muk/prism-v2/internal/tui/types"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderSubscription 渲染訂閱鏈接
func RenderSubscription(info *types.SubscriptionInfo, ti textinput.Model, statusMsg string) string {
	header := renderSubpageHeader("訂閱鏈接")

	desc := lipgloss.NewStyle().
		Foreground(style.Snow2).
		Render(" 在線/離線訂閱地址，支持主流客戶端")

	divider := lipgloss.NewStyle().
		Foreground(style.Polar4).
		Render(strings.Repeat("─", 50))

	labelStyle := lipgloss.NewStyle().Foreground(style.Snow3)
	valueStyle := lipgloss.NewStyle().Foreground(style.Aurora2)

	var content string
	if info != nil {
		// 在線訂閱
		onlineTitle := lipgloss.NewStyle().
			Foreground(style.StatusGreen).
			Bold(true).
			Render(" 📡 在線訂閱（推薦）")

		onlineDesc := labelStyle.Render(" 訂閱地址:")
		onlineURL := lipgloss.NewStyle().
			Foreground(style.Snow1).
			Render(info.OnlineURL)

		onlineTip := lipgloss.NewStyle().
			Foreground(style.Muted).
			Render(" 支持 V2RayN, Clash, Shadowrocket 等")

		// 離線訂閱
		offlineTitle := lipgloss.NewStyle().
			Foreground(style.Aurora3).
			Bold(true).
			Render(" 💾 離線訂閱")

		offlineDesc := labelStyle.Render(" 訂閱內容:")
		offlineURL := lipgloss.NewStyle().
			Foreground(style.Snow1).
			MaxWidth(50).
			Render(info.OfflineURL)

		offlineTip := lipgloss.NewStyle().
			Foreground(style.Muted).
			Render(" Base64 編碼，可直接導入客戶端")

		// 統計信息
		statsLine := fmt.Sprintf(" %s %s  %s %s",
			labelStyle.Render("節點數量:"),
			valueStyle.Render(fmt.Sprintf("%d", info.NodeCount)),
			labelStyle.Render("更新時間:"),
			lipgloss.NewStyle().Foreground(style.Snow3).Render(info.UpdateTime))

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			onlineTitle,
			onlineDesc,
			onlineURL,
			onlineTip,
			"",
			offlineTitle,
			offlineDesc,
			offlineURL,
			offlineTip,
			"",
			statsLine,
		)
	} else {
		content = lipgloss.NewStyle().
			Foreground(style.Muted).
			Render("正在生成訂閱鏈接...")
	}

	items := []MenuItem{
		{"", "", "", lipgloss.Color("")},
		{constants.KeySubscription_CopyOnline, "複製 在線訂閱", "(獲取 URL)", style.StatusGreen},
		{constants.KeySubscription_CopyOffline, "複製 離線內容", "(Base64 導入)", style.Snow1},
		{constants.KeySubscription_Refresh, "刷新 訂閱數據", "(重新生成)", style.Aurora2},
		{constants.KeySubscription_QRCode, "生成 訂閱二維碼", "(手機掃碼)", style.Snow1},
	}

	menu := renderMenuWithAlignment(items, 0, "", false)

	statusBlock := RenderStatusMessage(statusMsg)

	footer := RenderInputFooter(ti)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		desc,
		divider,
		content,
		menu,
		statusBlock,
		footer,
	)
}

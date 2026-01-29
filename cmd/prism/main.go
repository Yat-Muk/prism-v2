package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/Yat-Muk/prism-v2/internal/pkg/appctx"
	"github.com/Yat-Muk/prism-v2/internal/pkg/logger"
	"github.com/Yat-Muk/prism-v2/internal/pkg/version"
	"github.com/Yat-Muk/prism-v2/internal/tui/model"
	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
)

func main() {
	// 1. 命令行參數解析
	var (
		workDir   = flag.String("dir", "", "指定工作目錄 (默認: /etc/prism 或 ~/.prism)")
		cronMode  = flag.Bool("cron", false, "執行定時維護任務並退出")
		showVer   = flag.Bool("version", false, "顯示版本信息")
		debugFlag = flag.Bool("debug", false, "開啟調試模式")
	)
	flag.Parse()

	if *showVer {
		fmt.Println(version.Info())
		os.Exit(0)
	}

	// 2. 環境初始化
	paths, err := appctx.NewPaths(*workDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "致命錯誤: 無法初始化路徑: %v\n", err)
		os.Exit(1)
	}

	stdErrFile := filepath.Join(paths.LogDir, "stderr.log")
	redirectStdErr(stdErrFile)

	logConfig := logger.DefaultConfig()
	logConfig.OutputPath = filepath.Join(paths.LogDir, "prism.log")
	logConfig.Console = false
	if *debugFlag {
		logConfig.Level = "debug"
	}

	log, err := logger.New(logConfig)
	if err != nil {
		panic(fmt.Sprintf("日誌初始化失敗: %v", err))
	}
	defer log.Sync()

	log.Info("Prism 正在啟動",
		zap.String("version", version.Version),
		zap.String("commit", version.GitCommit),
		zap.Bool("cron_mode", *cronMode),
	)

	// 3. 依賴注入
	deps, err := initializeDependencies(log, paths)
	if err != nil {
		log.Fatal("依賴初始化失敗", zap.Error(err))
	}

	// 4. 模式分發
	if *cronMode {
		log.Info("進入自動維護模式")
		if err := runCronTask(context.Background(), log, deps); err != nil {
			log.Error("定時任務執行失敗", zap.Error(err))
			os.Exit(1)
		}
		log.Info("定時任務執行成功")
		return
	}

	runTUI(deps)
}

func runTUI(deps *AppDependencies) {
	// 初始化業務路由 (Router)
	router := model.NewRouter(deps.HandlerConfig)

	mainModel := model.NewModel(router)

	// 啟動 Bubble Tea
	p := tea.NewProgram(
		mainModel,
		tea.WithAltScreen(),
	)

	// 4. 崩潰保護
	defer func() {
		if r := recover(); r != nil {
			p.ReleaseTerminal()
			fmt.Printf("\n\n❌ 程序崩潰: %v\n", r)
			deps.Log.Error("Panic", zap.Any("error", r), zap.String("stack", string(debug.Stack())))
			os.Exit(1)
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("程序運行錯誤: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("👋 Bye! 服務正在後台運行。")
}

func redirectStdErr(filename string) {
	_ = os.MkdirAll(filepath.Dir(filename), 0755)
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err == nil {
		os.Stderr = f
	}
}

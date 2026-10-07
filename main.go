package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"go-markdown-my-project/app"
	"go-markdown-my-project/core/i18n"
	"go-markdown-my-project/core/logger"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 初始化日志（对齐 log4rs.yml：控制台 + 滚动文件）
	_ = logger.Init(logger.Options{Level: "debug", Console: true})
	logger.Info(i18n.T("log.starting"))

	appObj := app.NewApp()
	err := wails.Run(&options.App{
		Title:  "Project Docs GUI",
		Width:  1200,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        appObj.Startup,
		Bind: []interface{}{
			appObj,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}

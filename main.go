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
	title := "Project Docs GUI"
	if v := app.FormatBuildVersion(); v != "dev" { // 编译期注入的版本戳格式化为 (build YYYY-MM-DD HH:mm:ss)，dev 下不加后缀
		title += " " + v
	}
	err := wails.Run(&options.App{
		// 回退尺寸：取不到屏幕信息时使用；StartHidden 下由 app.App.Startup 按主屏 75% 设定并显示
		Title:       title,
		Width:       1200,
		Height:      800,
		MinWidth:    640,  // 下限放宽，避免小屏上把 75% 的窗口顶大
		MinHeight:   400,
		StartHidden: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// 与 --bg-0 (#14161a) 对齐，避免启动瞬间闪出偏蓝底色
		BackgroundColour: &options.RGBA{R: 20, G: 22, B: 26, A: 1},
		OnStartup:        appObj.Startup,
		Bind: []interface{}{
			appObj,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}

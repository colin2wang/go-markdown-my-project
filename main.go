package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

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

	wailsApp := application.New(application.Options{
		Name: "ProjectDocs",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Services: []application.Service{
			application.NewService(appObj),
		},
	})
	// 注入应用引用，供事件/对话框使用
	appObj.SetApplication(wailsApp)

	// v3 由主循环启动后正常显示窗口；Hidden + 手动 Show/Run 在 v3.1.0 下会卡死或不显示
	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     title,
		Width:     1400,
		Height:    800,
		MinWidth:  640, // 下限放宽，避免小屏上把 75% 的窗口顶大
		MinHeight: 400,
		// 与 --bg-0 (#14161a) 对齐，避免启动瞬间闪出偏蓝底色
		BackgroundType:   application.BackgroundTypeSolid,
		BackgroundColour: application.NewRGBA(20, 22, 26, 0xFF),
	})

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}

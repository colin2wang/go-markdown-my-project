// 版本信息：由 build.ps1 在编译时通过 -ldflags -X 注入本地时区的构建时间戳
// （如 2026-11-04_21:50:00，下划线占位避免 ldflags 值含空格），本地 go build / wails dev 下为 dev。
package app

import "strings"

// BuildVersion 构建版本戳，编译期注入，勿在代码里赋值。
var BuildVersion = "dev"

// FormatBuildVersion 返回展示用版本串："(build 2026-11-04 21:50:00)"；dev 下原样返回。
func FormatBuildVersion() string {
	if BuildVersion == "dev" {
		return BuildVersion
	}
	return "(build " + strings.ReplaceAll(BuildVersion, "_", " ") + ")"
}

// GetVersion 返回展示用版本串，供前端标题栏展示。
func (a *App) GetVersion() string {
	return FormatBuildVersion()
}

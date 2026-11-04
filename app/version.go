// 版本信息：由 build.ps1 在编译时通过 -ldflags -X 注入本地时区的构建时间戳
// （如 build-20261101-1530），本地 go build / wails dev 下为 dev。
package app

// BuildVersion 构建版本戳，编译期注入，勿在代码里赋值。
var BuildVersion = "dev"

// GetVersion 返回构建版本戳，供前端标题栏展示。
func (a *App) GetVersion() string {
	return BuildVersion
}

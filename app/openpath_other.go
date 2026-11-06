//go:build !windows

// 非 Windows 平台的「打开目录」：macOS 的 open 会激活并复用 Finder 窗口；
// Linux/其他由 xdg-open 按 DE 默认文件管理器处理，无需窗口复用逻辑。
package app

import (
	"os/exec"
	"path/filepath"
	"runtime"
)

// openFileInFileManager 在文件管理器中打开文件所在目录并选中该文件。
// macOS 的 open -R 会 reveal（选中）文件；Linux/其他无通用选中方式，
// 退化为打开文件所在目录。
func openFileInFileManager(path string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", "-R", path).Start()
	}
	return exec.Command("xdg-open", filepath.Dir(path)).Start()
}

// openInFileManager 打开目录。入参应为已 Abs 归一的目录路径。
func openInFileManager(dir string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", dir).Start()
	default:
		return exec.Command("xdg-open", dir).Start()
	}
}

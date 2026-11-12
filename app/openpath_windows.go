//go:build windows

// Shell COM（IShellWindows）复用资源管理器窗口：枚举当前已打开的资源管理器
// 窗口，若某窗口正显示目标目录则仅恢复/置前（复用），否则回退 explorer 新开。
// 任一 COM 步骤失败都静默回退，功能不退化。
package app

import (
	"net/url"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// Shell Windows 集合对象（资源管理器/IE 窗口枚举）
const clsidShellWindows = "{9BA05972-F6A8-11CF-A442-00A0C90A8F39}"

// S_FALSE：COM 已在当前线程初始化过（仍需配对 CoUninitialize）
const sFalse = 0x00000001

const swRestore = 9

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procIsIconic            = user32.NewProc("IsIconic")
	procShowWindow          = user32.NewProc("ShowWindow")
)

// openFileInFileManager 在文件管理器中打开文件所在目录并选中该文件
// （explorer /select，总是新开窗口）。Win10 下 /select 会复用已显示同目录的
// 窗口，Win11（标签页）则总是新开；为行为一致，先尝试复用父目录窗口，
// 命中则仅置前（不做选中高亮），未命中才走 /select 新开。
func openFileInFileManager(path string) error {
	if bringExplorerToFront(filepath.Dir(path)) {
		return nil
	}
	return exec.Command("explorer", "/select,", path).Start()
}

// openInFileManager 打开目录：优先复用已显示该目录的资源管理器窗口。
// 入参应为已 Abs 归一的目录路径（OpenPath 负责文件→父目录转换）。
func openInFileManager(dir string) error {
	if bringExplorerToFront(dir) {
		return nil
	}
	return exec.Command("explorer", dir).Start()
}

// bringExplorerToFront 在已打开的资源管理器窗口里找正显示 dir 的那个，
// 找到则恢复/置前并返回 true。
func bringExplorerToFront(dir string) bool {
	// COM 需绑定到 OS 线程且每线程初始化一次；调用点可能在任意 goroutine。
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// STA。S_FALSE（已初始化）按成功处理；其他错误（含宿主线程已是 MTA 的
	// RPC_E_CHANGED_MODE）放弃复用，回退新开窗口。
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		if code, ok := err.(*ole.OleError); !ok || code.Code() != sFalse {
			return false
		}
	}
	defer ole.CoUninitialize()

	unk, err := oleutil.CreateObject(clsidShellWindows)
	if err != nil {
		return false
	}
	defer unk.Release()

	disp, err := unk.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return false
	}
	defer disp.Release()

	countV, err := oleutil.GetProperty(disp, "Count")
	if err != nil {
		return false
	}
	count := int(countV.Val)

	want := normalizeDirPath(dir)
	for i := 0; i < count; i++ {
		itemV, err := oleutil.CallMethod(disp, "Item", i)
		if err != nil {
			continue // 窗口可能正在关闭，跳过
		}
		item := itemV.ToIDispatch()
		urlV, err := oleutil.GetProperty(item, "LocationURL")
		if err != nil {
			item.Release()
			continue
		}
		hwndV, err := oleutil.GetProperty(item, "HWND")
		if err != nil {
			item.Release()
			continue
		}
		hwnd := uintptr(hwndV.Val)
		isMatch := normalizeExplorerURL(urlV.ToString()) == want
		item.Release()
		if isMatch && hwnd != 0 {
			r1, _, _ := procIsIconic.Call(hwnd)
			if r1 != 0 { // 最小化则先恢复
				procShowWindow.Call(hwnd, swRestore)
			}
			procSetForegroundWindow.Call(hwnd)
			return true
		}
	}
	return false
}

// normalizeExplorerURL 把 LocationURL（file:///F:/a/b）规范化为与
// normalizeDirPath 可比较的形式。
func normalizeExplorerURL(raw string) string {
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(raw), "file://") {
		return "" // 虚拟目录（此电脑/回收站等）无文件 URL，不参与匹配
	}
	path := strings.TrimPrefix(raw[len("file://"):], "/")
	if dec, err := url.PathUnescape(path); err == nil {
		path = dec
	}
	return normalizeDirPath(path)
}

// normalizeDirPath 统一分隔符/大小写/尾部斜杠，用于窗口路径匹配。
func normalizeDirPath(p string) string {
	return strings.ToLower(strings.TrimRight(filepath.Clean(filepath.FromSlash(p)), string(filepath.Separator)))
}

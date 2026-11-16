//go:build windows

// Shell COM（IShellWindows）复用资源管理器窗口：枚举当前已打开的资源管理器
// 窗口，若某窗口/标签正显示目标目录则恢复/置前（复用），并经
// IShellFolderViewDual::SelectItem 在该视图内选中目标文件；未打开时回退
// explorer 命令行（/select 或直接打开）。任一 COM 步骤失败都静默回退，
// 功能不退化。
package app

import (
	"net/url"
	"os"
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

// IShellFolderViewDual::SelectItem 的 SVSI 标志（shobjidl_core.h）
const (
	svsiSelect         = 0x00000001
	svsiDeselectOthers = 0x00000004
	svsiEnsureVisible  = 0x00000008
	svsiFocused        = 0x00000010
	// 选中 + 取消其他选中 + 滚动到可见 + 设为焦点项
	svsiSelectOne = svsiSelect | svsiDeselectOthers | svsiEnsureVisible | svsiFocused
)

// S_FALSE：COM 已在当前线程初始化过（仍需配对 CoUninitialize）
const sFalse = 0x00000001

const swRestore = 9

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procIsIconic            = user32.NewProc("IsIconic")
	procShowWindow          = user32.NewProc("ShowWindow")
)

// openFileInFileManager 打开文件所在目录并选中该文件：优先复用已显示该目录的
// 窗口（复用 + SelectItem），否则 explorer /select 新开（/select 后面的逗号
// 必须保留，否则 Explorer 忽略选中、直接打开「文档」）。
func openFileInFileManager(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if reuseExplorerWindow(filepath.Dir(abs), abs) {
		return nil
	}
	return exec.Command("explorer", "/select,", abs).Start()
}

// openInFileManager 打开目录：已打开 → 复用窗口并选中第一个文件；
// 未打开 → 新开窗口并选中；空目录 → 复用或直接打开。
// 入参应为已 Abs 归一的目录路径（OpenPath 负责文件→父目录转换）。
func openInFileManager(dir string) error {
	first := firstFileIn(dir)
	if first != "" {
		if reuseExplorerWindow(dir, first) {
			return nil
		}
		return exec.Command("explorer", "/select,", first).Start()
	}
	// 空目录：无文件可选
	if bringExplorerToFront(dir) {
		return nil
	}
	return exec.Command("explorer", dir).Start()
}

// firstFileIn 按名称排序取目录内第一个文件（跳过子目录）。
func firstFileIn(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}

// reuseExplorerWindow 遍历 ShellWindows（Win11 下每个标签是独立条目），
// 命中 LocationURL 后还原/置前窗口，并在该视图内选中目标文件。
func reuseExplorerWindow(dir, selectFile string) bool {
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

	want := normalizeDirPath(dir)
	selectName := strings.ToLower(filepath.Base(selectFile))
	for i := 0; i < int(countV.Val); i++ {
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
		if normalizeExplorerURL(urlV.ToString()) != want || hwnd == 0 {
			item.Release()
			continue
		}
		// ① 还原（若最小化）+ 置前
		if r1, _, _ := procIsIconic.Call(hwnd); r1 != 0 {
			procShowWindow.Call(hwnd, swRestore)
		}
		procSetForegroundWindow.Call(hwnd)
		// ② 视图内选中（失败不影响「复用窗口」这个结果）
		selectInWindow(item, selectName)
		item.Release()
		return true
	}
	return false
}

// selectInWindow 经 Document（IShellFolderViewDual）在已打开的视图中选中文件。
// 按名称精确匹配而非取 Items.Item(0)：Explorer 视图默认「文件夹排前」，
// Item(0) 可能是子目录。
func selectInWindow(item *ole.IDispatch, lowerName string) {
	docV, err := oleutil.GetProperty(item, "Document")
	if err != nil {
		return
	}
	doc := docV.ToIDispatch()
	if doc == nil {
		return
	}
	defer doc.Release()

	folderV, err := oleutil.GetProperty(doc, "Folder")
	if err != nil {
		return
	}
	folder := folderV.ToIDispatch()
	if folder == nil {
		return
	}
	defer folder.Release()

	itemsV, err := oleutil.CallMethod(folder, "Items")
	if err != nil {
		return
	}
	items := itemsV.ToIDispatch()
	if items == nil {
		return
	}
	defer items.Release()

	countV, err := oleutil.GetProperty(items, "Count")
	if err != nil {
		return
	}

	var target *ole.IDispatch
	for i := 0; i < int(countV.Val) && target == nil; i++ {
		itV, err := oleutil.CallMethod(items, "Item", i)
		if err != nil {
			continue
		}
		it := itV.ToIDispatch()
		if it == nil {
			continue
		}
		nameV, err := oleutil.GetProperty(it, "Name")
		if err != nil {
			it.Release()
			continue
		}
		if strings.ToLower(nameV.ToString()) == lowerName {
			target = it // 命中：所有权转交 target，不再 Release
		} else {
			it.Release()
		}
	}
	if target == nil {
		return
	}
	defer target.Release()

	oleutil.CallMethod(doc, "SelectItem", target, svsiSelectOne)
}

// bringExplorerToFront 在已打开的资源管理器窗口里找正显示 dir 的那个，
// 找到则恢复/置前并返回 true（空目录等无文件可选的场景使用）。
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

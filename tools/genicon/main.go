// Command genicon 把 assets/icon.svg 转换为应用图标（PNG + ICO）。
//
// 用法：
//
//	go run ./tools/genicon                 # 使用默认路径与尺寸
//	go run ./tools/genicon -size 256       # 指定 PNG 输出尺寸
//	go run ./tools/genicon -ico ""         # 只输出 PNG
//
// 仅依赖标准库：自行解析 SVG 子集并用扫描线算法栅格化，避免引入第三方矢量库。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	in := flag.String("in", "assets/icon.svg", "输入 SVG 文件")
	pngOut := flag.String("png", "build/appicon.png", "输出 PNG 文件（空串 = 不输出）")
	icoOut := flag.String("ico", "build/windows/icon.ico", "输出 ICO 文件（空串 = 不输出）")
	size := flag.Int("size", 512, "PNG 输出边长（像素）")
	sizes := flag.String("sizes", "16,24,32,48,64,128,256", "ICO 内嵌尺寸列表")
	flag.Parse()

	doc, err := ParseSVGFile(*in)
	if err != nil {
		fail(err)
	}
	if len(doc.Shapes) == 0 {
		fail(fmt.Errorf("SVG 中没有可绘制图元: %s", *in))
	}

	if *pngOut != "" {
		if err := writePNG(doc, *pngOut, *size); err != nil {
			fail(err)
		}
		fmt.Printf("[ OK ] png %s (%dx%d)\n", *pngOut, *size, *size)
	}
	if *icoOut != "" {
		list, err := parseSizes(*sizes)
		if err != nil {
			fail(err)
		}
		if err := writeICO(doc, *icoOut, list); err != nil {
			fail(err)
		}
		fmt.Printf("[ OK ] ico %s (%s)\n", *icoOut, strings.Join(sizeNames(list), ","))
	}
	fmt.Printf("[ OK ] %d 个图元，viewBox %.0fx%.0f\n", len(doc.Shapes), doc.Width, doc.Height)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "[FAIL]", err)
	os.Exit(1)
}

func parseSizes(s string) ([]int, error) {
	var out []int
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n <= 0 || n > 256 {
			return nil, fmt.Errorf("无效的图标尺寸: %q", p)
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("图标尺寸列表为空")
	}
	return out, nil
}

func sizeNames(sizes []int) []string {
	out := make([]string, 0, len(sizes))
	for _, s := range sizes {
		out = append(out, strconv.Itoa(s))
	}
	return out
}

func writePNG(doc *Document, path string, size int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return EncodePNG(path, Render(doc, size, size))
}

func writeICO(doc *Document, path string, sizes []int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return EncodeICO(path, doc, sizes)
}

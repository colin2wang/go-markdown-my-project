// Package main 实现 SVG → PNG/ICO 的图标转换（受限 SVG 子集，纯标准库栅格化）。
//
// 支持的 SVG 子集：svg/defs/g、linearGradient+stop、rect(rx/ry)、circle、
// ellipse、line、polyline、polygon、path（M/L/H/V/C/S/Q/T/A/Z 及其相对形式），
// 呈现属性 fill/stroke/stroke-width/opacity/fill-opacity/fill-rule/transform，
// 不支持 clip/mask/text/滤镜，遇到未知图元会直接跳过（不做静默错绘）。
package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Point 二维点（SVG 用户坐标）。
type Point struct{ X, Y float64 }

// Matrix 2D 仿射矩阵：x' = A*x + C*y + E，y' = B*x + D*y + F。
type Matrix struct{ A, B, C, D, E, F float64 }

// Identity 单位矩阵。
func Identity() Matrix { return Matrix{A: 1, D: 1} }

// Mul 返回复合矩阵：先应用 n 再应用 m。
func (m Matrix) Mul(n Matrix) Matrix {
	return Matrix{
		A: m.A*n.A + m.C*n.B,
		B: m.B*n.A + m.D*n.B,
		C: m.A*n.C + m.C*n.D,
		D: m.B*n.C + m.D*n.D,
		E: m.A*n.E + m.C*n.F + m.E,
		F: m.B*n.E + m.D*n.F + m.F,
	}
}

// Apply 变换一个点。
func (m Matrix) Apply(p Point) Point {
	return Point{X: m.A*p.X + m.C*p.Y + m.E, Y: m.B*p.X + m.D*p.Y + m.F}
}

// Invert 求逆矩阵；奇异矩阵返回 Identity。
func (m Matrix) Invert() Matrix {
	det := m.A*m.D - m.B*m.C
	if det == 0 {
		return Identity()
	}
	return Matrix{
		A: m.D / det, B: -m.B / det,
		C: -m.C / det, D: m.A / det,
		E: (m.C*m.F - m.D*m.E) / det,
		F: (m.B*m.E - m.A*m.F) / det,
	}
}

// Color 颜色分量，各通道 0..1（已在 sRGB 空间，本工具不做色彩管理）。
type Color struct{ R, G, B, A float64 }

// Paint 填充或描边使用的颜料。
type Paint struct {
	None bool      // fill="none"
	Col  Color     // 纯色
	Grad *Gradient // 渐变（不为 nil 时优先）
}

// Gradient 线性渐变（仅 linearGradient，支持多停靠点）。
type Gradient struct {
	ID        string
	X1, Y1    float64 // gradientUnits=userSpaceOnUse 时为坐标；objectBoundingBox 时为 0..1
	X2, Y2    float64
	UserSpace bool
	Stops     []Stop
}

// Stop 渐变停靠点。
type Stop struct {
	Offset float64
	Color  Color
}

// GradientAt 在归一化参数 t（0..1）处取色。
func (g *Gradient) At(t float64) Color {
	if len(g.Stops) == 0 {
		return Color{}
	}
	if t <= g.Stops[0].Offset {
		return g.Stops[0].Color
	}
	last := g.Stops[len(g.Stops)-1]
	if t >= last.Offset {
		return last.Color
	}
	for i := 1; i < len(g.Stops); i++ {
		a, b := g.Stops[i-1], g.Stops[i]
		if t <= b.Offset {
			span := b.Offset - a.Offset
			if span <= 0 {
				return b.Color
			}
			k := (t - a.Offset) / span
			return lerpColor(a.Color, b.Color, k)
		}
	}
	return last.Color
}

func lerpColor(a, b Color, k float64) Color {
	return Color{
		R: a.R + (b.R-a.R)*k,
		G: a.G + (b.G-a.G)*k,
		B: a.B + (b.B-a.B)*k,
		A: a.A + (b.A-a.A)*k,
	}
}

// Shape 一个可绘制图元：轮廓与描边折线均在 SVG 用户坐标下。
type Shape struct {
	Contours [][]Point // 闭合填充轮廓
	Strokes  [][]Point // 描边折线（按子路径；闭合子路径首尾点相同）
	Fill     Paint
	Stroke   Paint
	Width    float64 // 描边宽度（用户坐标）
	EvenOdd  bool
	Opacity  float64
	M        Matrix // 用户坐标 → viewBox 坐标（含祖先 transform）
}

// Document 解析后的 SVG 文档。
type Document struct {
	Width, Height float64 // viewBox 尺寸
	Shapes        []Shape
	Gradients     map[string]*Gradient
}

// node 通用 XML 节点（由 parseXMLTree 手工构建；encoding/xml 的 `,anyattr`
// 标签在部分 Go 版本下不生效，故不走 xml.Unmarshal）。
type node struct {
	XMLName xml.Name
	Attrs   []xml.Attr
	Nodes   []node
}

// parseXMLTree 用 xml.Decoder 逐 token 构建 node 树（自闭合元素会立即产生
// 配对的 EndElement，无需特判）。
func parseXMLTree(data []byte) (*node, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var stack []*node
	var root *node
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			stack = append(stack, &node{XMLName: t.Name, Attrs: t.Attr})
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, fmt.Errorf("XML 标签不配对: </%s>", t.Name.Local)
			}
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				root = n
			} else {
				parent := stack[len(stack)-1]
				parent.Nodes = append(parent.Nodes, *n)
			}
		}
	}
	if root == nil {
		return nil, fmt.Errorf("XML 内容为空")
	}
	return root, nil
}

// ParseSVGFile 读取并解析 SVG 文件。
func ParseSVGFile(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 SVG 失败: %w", err)
	}
	return ParseSVG(data)
}

// ParseSVG 解析 SVG 文本。
func ParseSVG(data []byte) (*Document, error) {
	root, err := parseXMLTree(data)
	if err != nil {
		return nil, fmt.Errorf("解析 SVG 失败: %w", err)
	}
	local := root.XMLName.Local
	if local != "svg" {
		return nil, fmt.Errorf("根节点不是 <svg>: %s", local)
	}
	doc := &Document{Gradients: map[string]*Gradient{}}
	vb := attr(*root, "viewBox")
	if vb != "" {
		f := parseFloatList(vb)
		if len(f) == 4 {
			doc.Width, doc.Height = f[2], f[3]
		}
	}
	if doc.Width <= 0 || doc.Height <= 0 {
		doc.Width = parseFloat(attr(*root, "width"), 512)
		doc.Height = parseFloat(attr(*root, "height"), 512)
	}
	// 先收集 defs 中的渐变（defs 位置可能在引用之后）
	collectDefs(root, doc)
	st := style{}
	st.Fill = Paint{Col: Color{A: 1}}
	st.Stroke = Paint{None: true}
	st.Opacity = 1
	walk(doc, root, st, Identity())
	return doc, nil
}

// collectDefs 递归收集 <defs> 下的 linearGradient。
func collectDefs(n *node, doc *Document) {
	if n.XMLName.Local == "linearGradient" {
		g := &Gradient{ID: attr(*n, "id")}
		units := attr(*n, "gradientUnits")
		g.UserSpace = units == "userSpaceOnUse"
		g.X1 = parseFloat(attr(*n, "x1"), 0)
		g.Y1 = parseFloat(attr(*n, "y1"), 0)
		g.X2 = parseFloat(attr(*n, "x2"), 1)
		g.Y2 = parseFloat(attr(*n, "y2"), 0)
		for i := range n.Nodes {
			s := &n.Nodes[i]
			if s.XMLName.Local != "stop" {
				continue
			}
			off := parseFloat(attr(*s, "offset"), 0)
			if strings.HasSuffix(strings.TrimSpace(attr(*s, "offset")), "%") {
				off /= 100
			}
			g.Stops = append(g.Stops, Stop{Offset: off, Color: parseColor(attr(*s, "stop-color"), Color{A: 1})})
		}
		if len(g.Stops) == 0 {
			g.Stops = []Stop{{Offset: 0, Color: Color{A: 1}}, {Offset: 1, Color: Color{A: 1}}}
		}
		if g.ID != "" {
			doc.Gradients[g.ID] = g
		}
	}
	for i := range n.Nodes {
		collectDefs(&n.Nodes[i], doc)
	}
}

// style 可继承的呈现属性。
type style struct {
	Fill     Paint
	Stroke   Paint
	Width    float64
	Opacity  float64
	EvenOdd  bool
	FillRule string
}

// walk 递归遍历元素树，生成 Shape。
func walk(doc *Document, n *node, parent style, m Matrix) {
	// <defs> 不参与绘制；其余容器递归
	if n.XMLName.Local == "defs" || n.XMLName.Local == "linearGradient" {
		return
	}
	st := inherit(parent, *n, doc)
	cm := m.Mul(parseTransform(attr(*n, "transform")))

	added := false
	switch n.XMLName.Local {
	case "rect":
		added = addRect(doc, st, cm, *n)
	case "circle":
		added = addCircle(doc, st, cm, *n)
	case "ellipse":
		added = addEllipse(doc, st, cm, *n)
	case "line":
		added = addLine(doc, st, cm, *n)
	case "polyline", "polygon":
		added = addPoly(doc, st, cm, *n)
	case "path":
		added = addPath(doc, st, cm, *n)
	}
	_ = added
	for i := range n.Nodes {
		walk(doc, &n.Nodes[i], st, cm)
	}
}

// inherit 合并元素自身呈现属性到继承样式。
func inherit(p style, n node, doc *Document) style {
	st := p
	a := attrMap(n)
	applyStyleAttr(&st, a["style"], doc)
	if v, ok := a["fill"]; ok {
		st.Fill = parsePaint(v, doc)
	}
	if v, ok := a["stroke"]; ok {
		st.Stroke = parsePaint(v, doc)
	}
	if v, ok := a["stroke-width"]; ok {
		st.Width = parseFloat(v, st.Width)
	}
	if v, ok := a["opacity"]; ok {
		st.Opacity = parseFloat(v, 1)
	}
	if v, ok := a["fill-rule"]; ok {
		st.FillRule = v
	}
	st.EvenOdd = strings.EqualFold(st.FillRule, "evenodd")
	return st
}

// applyStyleAttr 解析 style="a:b;c:d" 形式（优先级低于同名呈现属性）。
func applyStyleAttr(st *style, s string, doc *Document) {
	if s == "" {
		return
	}
	for _, part := range strings.Split(s, ";") {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		switch k {
		case "fill":
			st.Fill = parsePaint(v, doc)
		case "stroke":
			st.Stroke = parsePaint(v, doc)
		case "stroke-width":
			st.Width = parseFloat(v, st.Width)
		case "opacity":
			st.Opacity = parseFloat(v, 1)
		case "fill-rule":
			st.FillRule = v
			st.EvenOdd = strings.EqualFold(v, "evenodd")
		}
	}
}

// push 生成 Shape 并追加到文档（填充与描边均无内容时跳过）。
func push(doc *Document, st style, m Matrix, contours [][]Point, strokes [][]Point) {
	if len(contours) == 0 && len(strokes) == 0 {
		return
	}
	if st.Opacity <= 0 {
		return
	}
	doc.Shapes = append(doc.Shapes, Shape{
		Contours: contours,
		Strokes:  strokes,
		Fill:     st.Fill,
		Stroke:   st.Stroke,
		Width:    st.Width,
		EvenOdd:  st.EvenOdd,
		Opacity:  st.Opacity,
		M:        m,
	})
}

func addRect(doc *Document, st style, m Matrix, n node) bool {
	a := attrMap(n)
	x := parseFloat(a["x"], 0)
	y := parseFloat(a["y"], 0)
	w := parseFloat(a["width"], 0)
	h := parseFloat(a["height"], 0)
	if w <= 0 || h <= 0 {
		return false
	}
	rx := parseFloat(a["rx"], -1)
	ry := parseFloat(a["ry"], -1)
	if rx < 0 && ry < 0 {
		rx, ry = 0, 0
	} else if rx < 0 {
		rx = ry
	} else if ry < 0 {
		ry = rx
	}
	if rx > w/2 {
		rx = w / 2
	}
	if ry > h/2 {
		ry = h / 2
	}
	push(doc, st, m, [][]Point{roundRect(x, y, w, h, rx, ry)}, nil)
	return true
}

func addCircle(doc *Document, st style, m Matrix, n node) bool {
	a := attrMap(n)
	cx, cy := parseFloat(a["cx"], 0), parseFloat(a["cy"], 0)
	r := parseFloat(a["r"], 0)
	if r <= 0 {
		return false
	}
	push(doc, st, m, [][]Point{ellipseContour(cx, cy, r, r)}, nil)
	return true
}

func addEllipse(doc *Document, st style, m Matrix, n node) bool {
	a := attrMap(n)
	cx, cy := parseFloat(a["cx"], 0), parseFloat(a["cy"], 0)
	rx, ry := parseFloat(a["rx"], 0), parseFloat(a["ry"], 0)
	if rx <= 0 || ry <= 0 {
		return false
	}
	push(doc, st, m, [][]Point{ellipseContour(cx, cy, rx, ry)}, nil)
	return true
}

func addLine(doc *Document, st style, m Matrix, n node) bool {
	a := attrMap(n)
	x1, y1 := parseFloat(a["x1"], 0), parseFloat(a["y1"], 0)
	x2, y2 := parseFloat(a["x2"], 0), parseFloat(a["y2"], 0)
	push(doc, st, m, nil, [][]Point{{Point{x1, y1}, Point{x2, y2}}})
	return true
}

func addPoly(doc *Document, st style, m Matrix, n node) bool {
	a := attrMap(n)
	pts := parsePoints(a["points"])
	if len(pts) < 2 {
		return false
	}
	if n.XMLName.Local == "polygon" {
		pts = append(pts, pts[0])
		push(doc, st, m, [][]Point{pts}, nil)
		return true
	}
	push(doc, st, m, nil, [][]Point{pts})
	return true
}

func addPath(doc *Document, st style, m Matrix, n node) bool {
	d := attr(n, "d")
	if strings.TrimSpace(d) == "" {
		return false
	}
	subs, err := ParsePath(d)
	if err != nil {
		fmt.Fprintf(os.Stderr, "genicon: 跳过无法解析的 path: %v\n", err)
		return false
	}
	var contours, strokes [][]Point
	for _, sub := range subs {
		if len(sub) < 2 {
			continue
		}
		closed := sub[0].X == sub[len(sub)-1].X && sub[0].Y == sub[len(sub)-1].Y
		if closed {
			contours = append(contours, sub)
			if !st.Stroke.None && st.Width > 0 {
				strokes = append(strokes, sub)
			}
		} else {
			if !st.Fill.None {
				// 开放子路径仍参与填充（隐式闭合）
				contours = append(contours, append(append([]Point{}, sub...), sub[0]))
			}
			if !st.Stroke.None && st.Width > 0 {
				strokes = append(strokes, sub)
			}
		}
	}
	push(doc, st, m, contours, strokes)
	return true
}

// —— 几何辅助 ——

func roundRect(x, y, w, h, rx, ry float64) []Point {
	if rx <= 0 || ry <= 0 {
		return []Point{{x, y}, {x + w, y}, {x + w, y + h}, {x, y + h}}
	}
	var pts []Point
	// 上边 → 右上圆角（270°→0°）→ 右边 → 右下圆角（0°→90°）
	// → 下边 → 左下圆角（90°→180°）→ 左边 → 左上圆角（180°→270°）
	arc := func(cx, cy, from float64) {
		steps := 12
		for i := 0; i <= steps; i++ {
			t := from + (math_Pi/2)*float64(i)/float64(steps)
			pts = append(pts, Point{X: cx + rx*cos(t), Y: cy + ry*sin(t)}) //nolint
		}
	}
	pts = append(pts, Point{x + rx, y})
	pts = append(pts, Point{x + w - rx, y})
	arc(x+w-rx, y+ry, -math_Pi/2)
	pts = append(pts, Point{x + w, y + h - ry})
	arc(x+w-rx, y+h-ry, 0)
	pts = append(pts, Point{x + rx, y + h})
	arc(x+rx, y+h-ry, math_Pi/2)
	pts = append(pts, Point{x, y + ry})
	arc(x+rx, y+ry, math_Pi)
	return pts
}

func ellipseContour(cx, cy, rx, ry float64) []Point {
	const steps = 64
	pts := make([]Point, 0, steps+1)
	for i := 0; i <= steps; i++ {
		t := 2 * math_Pi * float64(i) / steps
		pts = append(pts, Point{X: cx + rx*cos(t), Y: cy + ry*sin(t)})
	}
	return pts
}

// —— 属性与数值解析 ——

func attrMap(n node) map[string]string {
	m := make(map[string]string, len(n.Attrs))
	for _, a := range n.Attrs {
		m[a.Name.Local] = a.Value
	}
	return m
}

func attr(n node, name string) string {
	for _, a := range n.Attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func parseFloat(s string, def float64) float64 {
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "px"))
	if s == "" {
		return def
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}

func parseFloatList(s string) []float64 {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == ',' || r == '\t' || r == '\n' || r == '\r'
	})
	out := make([]float64, 0, len(parts))
	for _, p := range parts {
		if v, err := strconv.ParseFloat(p, 64); err == nil {
			out = append(out, v)
		}
	}
	return out
}

func parsePoints(s string) []Point {
	f := parseFloatList(s)
	out := make([]Point, 0, len(f)/2)
	for i := 0; i+1 < len(f); i += 2 {
		out = append(out, Point{f[i], f[i+1]})
	}
	return out
}

// parsePaint 解析 fill/stroke 取值：none | #rgb | #rrggbb | rgb(...) | 命名色 | url(#id)。
func parsePaint(v string, doc *Document) Paint {
	v = strings.TrimSpace(v)
	switch {
	case v == "" || v == "none":
		return Paint{None: true}
	case strings.HasPrefix(v, "url("):
		id := strings.Trim(strings.TrimSuffix(strings.TrimPrefix(v, "url("), ")"), "# \t")
		if doc != nil {
			if g, ok := doc.Gradients[id]; ok {
				return Paint{Grad: g}
			}
		}
		return Paint{None: true}
	default:
		return Paint{Col: parseColor(v, Color{A: 1})}
	}
}

// namedColors 少量常用命名色（够用即可，缺失时回退黑色）。
var namedColors = map[string]Color{
	"white": {1, 1, 1, 1},
	"black": {0, 0, 0, 1},
	"red":   {1, 0, 0, 1},
	"green": {0, 0.5, 0, 1},
	"blue":  {0, 0, 1, 1},
	"none":  {0, 0, 0, 0},
}

func parseColor(v string, def Color) Color {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return def
	}
	if c, ok := namedColors[v]; ok {
		return c
	}
	if strings.HasPrefix(v, "#") {
		hex := v[1:]
		if len(hex) == 3 {
			hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		}
		if len(hex) != 6 {
			return def
		}
		n, err := strconv.ParseUint(hex, 16, 32)
		if err != nil {
			return def
		}
		return Color{
			R: float64((n>>16)&0xFF) / 255,
			G: float64((n>>8)&0xFF) / 255,
			B: float64(n&0xFF) / 255,
			A: 1,
		}
	}
	if strings.HasPrefix(v, "rgb(") {
		f := parseFloatList(strings.TrimSuffix(strings.TrimPrefix(v, "rgb("), ")"))
		if len(f) >= 3 {
			return Color{R: f[0] / 255, G: f[1] / 255, B: f[2] / 255, A: 1}
		}
	}
	return def
}

// parseTransform 解析 transform 属性（translate/scale/rotate/matrix，可组合）。
func parseTransform(s string) Matrix {
	m := Identity()
	s = strings.TrimSpace(s)
	if s == "" {
		return m
	}
	tokens := tokenizeTransform(s)
	for i := 0; i < len(tokens); {
		name := tokens[i]
		i++
		var args []float64
		for i < len(tokens) && !isTransformName(tokens[i]) {
			args = append(args, parseFloat(tokens[i], 0))
			i++
		}
		m = m.Mul(transformOp(name, args))
	}
	return m
}

func isTransformName(s string) bool {
	switch s {
	case "translate", "scale", "rotate", "matrix", "skewX", "skewY":
		return true
	}
	return false
}

func transformOp(name string, a []float64) Matrix {
	f := func(i int) float64 {
		if i < len(a) {
			return a[i]
		}
		return 0
	}
	switch name {
	case "translate":
		tx := f(0)
		ty := f(1)
		if len(a) == 1 {
			ty = 0
		}
		return Matrix{A: 1, D: 1, E: tx, F: ty}
	case "scale":
		sx, sy := f(0), f(1)
		if len(a) == 1 {
			sy = sx
		}
		return Matrix{A: sx, D: sy}
	case "rotate":
		ang := f(0) * math_Pi / 180
		cx, cy := cos(ang), sin(ang)
		r := Matrix{A: cx, B: cy, C: -cy, D: cx}
		if len(a) >= 3 {
			// rotate(a cx cy) = translate(cx,cy) rotate(a) translate(-cx,-cy)
			px, py := f(1), f(2)
			return Matrix{A: 1, D: 1, E: px, F: py}.Mul(r).Mul(Matrix{A: 1, D: 1, E: -px, F: -py})
		}
		return r
	case "matrix":
		if len(a) < 6 {
			return Identity()
		}
		return Matrix{A: a[0], B: a[1], C: a[2], D: a[3], E: a[4], F: a[5]}
	}
	return Identity()
}

// tokenizeTransform 把 "translate(1,2) scale(3)" 拆成 [translate 1 2 scale 3]。
func tokenizeTransform(s string) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if strings.TrimSpace(cur.String()) != "" {
			out = append(out, strings.TrimSpace(cur.String()))
		}
		cur.Reset()
	}
	for _, r := range s {
		switch r {
		case '(', ')', ',', ' ', '\t', '\n':
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

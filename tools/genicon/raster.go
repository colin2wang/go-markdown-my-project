package main

import (
	"image"
	"math"
	"sort"
)

// viewMatrix 计算 viewBox → 目标画布的变换（preserveAspectRatio 默认 xMidYMid meet）。
func viewMatrix(doc *Document, w, h int) Matrix {
	if doc.Width <= 0 || doc.Height <= 0 {
		return Identity()
	}
	s := math.Min(float64(w)/doc.Width, float64(h)/doc.Height)
	tx := (float64(w) - doc.Width*s) / 2
	ty := (float64(h) - doc.Height*s) / 2
	return Matrix{A: s, D: s, E: tx, F: ty}
}

// Render 把文档栅格化为 RGBA 图像（4 倍超采样 → 盒式下采样抗锯齿）。
func Render(doc *Document, w, h int) *image.RGBA {
	ss := 4
	if w < 96 {
		ss = 8 // 小尺寸图标提高超采样倍率，避免锯齿
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	mw, mh := w*ss, h*ss
	mask := make([]byte, mw*mh)
	base := viewMatrix(doc, mw, mh)

	for _, s := range doc.Shapes {
		m := base.Mul(s.M)
		inv := m.Invert()
		scale := math.Sqrt(math.Abs(m.A*m.D - m.B*m.C))

		if !s.Fill.None && len(s.Contours) > 0 {
			dev := transformContours(s.Contours, m)
			clearMask(mask)
			fillContours(mask, mw, mh, dev, s.EvenOdd)
			composite(img, mask, mw, ss, toImageBBox(bboxOf(dev, 0), ss), s.Fill, s.Opacity, inv, userBBox(s.Contours))
		}
		if !s.Stroke.None && s.Width > 0 && len(s.Strokes) > 0 {
			dev := transformContours(s.Strokes, m)
			hw := s.Width * scale * 0.5
			polys := strokeParts(dev, hw)
			clearMask(mask)
			fillContours(mask, mw, mh, polys, false)
			composite(img, mask, mw, ss, toImageBBox(bboxOf(dev, hw), ss), s.Stroke, s.Opacity, inv, userBBox(s.Strokes))
		}
	}
	return img
}

func clearMask(m []byte) {
	for i := range m {
		m[i] = 0
	}
}

func transformContours(src [][]Point, m Matrix) [][]Point {
	out := make([][]Point, 0, len(src))
	for _, c := range src {
		pts := make([]Point, 0, len(c))
		for _, p := range c {
			pts = append(pts, m.Apply(p))
		}
		out = append(out, pts)
	}
	return out
}

// xHit 扫描线与边的交点。
type xHit struct {
	x float64
	w int
}

// fillContours 扫描线填充：nonzero（默认）或 even-odd 规则。
// 坐标单位为子像素，采用上半开区间 [y0, y1) 判定避免顶点重复计数。
func fillContours(mask []byte, mw, mh int, contours [][]Point, evenOdd bool) {
	type edge struct {
		x0, y0, x1, y1 float64
		w              int
	}
	var edges []edge
	for _, c := range contours {
		n := len(c)
		if n < 2 {
			continue
		}
		for i := 0; i < n; i++ {
			a, b := c[i], c[(i+1)%n]
			if a.Y == b.Y {
				continue
			}
			w := 1
			if b.Y < a.Y {
				w = -1
			}
			edges = append(edges, edge{a.X, a.Y, b.X, b.Y, w})
		}
	}
	if len(edges) == 0 {
		return
	}
	xs := make([]xHit, 0, 64)
	for y := 0; y < mh; y++ {
		yc := float64(y) + 0.5
		xs = xs[:0]
		for _, e := range edges {
			lo, hi := e.y0, e.y1
			if lo > hi {
				lo, hi = hi, lo
			}
			if yc < lo || yc >= hi {
				continue
			}
			t := (yc - e.y0) / (e.y1 - e.y0)
			xs = append(xs, xHit{x: e.x0 + t*(e.x1-e.x0), w: e.w})
		}
		if len(xs) < 2 {
			continue
		}
		sort.Slice(xs, func(i, j int) bool { return xs[i].x < xs[j].x })
		wind := 0
		row := y * mw
		for i := 0; i+1 < len(xs); i++ {
			wind += xs[i].w
			inside := wind != 0
			if evenOdd {
				inside = wind%2 != 0
			}
			if !inside {
				continue
			}
			start := int(math.Ceil(xs[i].x - 0.5))
			end := int(math.Ceil(xs[i+1].x - 0.5))
			if start < 0 {
				start = 0
			}
			if end > mw {
				end = mw
			}
			for x := start; x < end; x++ {
				mask[row+x] = 1
			}
		}
	}
}

// strokeParts 把描边折线展平为同向（有向面积 > 0）的填充片元：
// 每段一个四边形 + 每个顶点一个圆（round join/cap），保证 nonzero 下不互相抵消。
func strokeParts(polys [][]Point, hw float64) [][]Point {
	if hw <= 0 {
		return nil
	}
	var out [][]Point
	for _, sub := range polys {
		n := len(sub)
		if n < 2 {
			continue
		}
		closed := sub[0].X == sub[n-1].X && sub[0].Y == sub[n-1].Y
		for i := 0; i+1 < n; i++ {
			a, b := sub[i], sub[i+1]
			dx, dy := b.X-a.X, b.Y-a.Y
			len2 := math.Sqrt(dx*dx + dy*dy)
			if len2 == 0 {
				continue
			}
			nx, ny := -dy/len2*hw, dx/len2*hw
			out = append(out, orientCCW([]Point{
				{a.X + nx, a.Y + ny}, {b.X + nx, b.Y + ny},
				{b.X - nx, b.Y - ny}, {a.X - nx, a.Y - ny},
			}))
		}
		from, to := 0, n
		if closed {
			from, to = 0, n-1
		}
		for i := from; i < to; i++ {
			out = append(out, circlePoly(sub[i], hw, 10))
		}
	}
	return out
}

// circlePoly 生成以 c 为中心、半径 r 的圆（有向面积为正）。
func circlePoly(c Point, r float64, steps int) []Point {
	pts := make([]Point, 0, steps)
	for i := 0; i < steps; i++ {
		t := 2 * math.Pi * float64(i) / float64(steps)
		pts = append(pts, Point{c.X + r*math.Cos(t), c.Y + r*math.Sin(t)})
	}
	return orientCCW(pts)
}

// orientCCW 统一顶点方向，使有向面积为正（避免 nonzero 填充互相抵消）。
func orientCCW(p []Point) []Point {
	a := signedArea(p)
	if a < 0 {
		for i, j := 0, len(p)-1; i < j; i, j = i+1, j-1 {
			p[i], p[j] = p[j], p[i]
		}
	}
	return p
}

func signedArea(p []Point) float64 {
	s := 0.0
	for i := range p {
		j := (i + 1) % len(p)
		s += p[i].X*p[j].Y - p[j].X*p[i].Y
	}
	return s / 2
}

// bbox 整数像素包围盒（右开区间）。
type bbox struct{ x0, y0, x1, y1 int }

func bboxOf(dev [][]Point, pad float64) bbox {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, c := range dev {
		for _, p := range c {
			if p.X < minX {
				minX = p.X
			}
			if p.Y < minY {
				minY = p.Y
			}
			if p.X > maxX {
				maxX = p.X
			}
			if p.Y > maxY {
				maxY = p.Y
			}
		}
	}
	if math.IsInf(minX, 1) {
		return bbox{}
	}
	return bbox{
		x0: int(math.Floor((minX - pad))),
		y0: int(math.Floor((minY - pad))),
		x1: int(math.Ceil(maxX + pad)),
		y1: int(math.Ceil(maxY + pad)),
	}
}

// toImageBBox 把子采样坐标的包围盒换算到最终图像像素坐标（右开区间）。
// composite 以图像像素迭代，而 bboxOf 产生的是超采样设备坐标，必须先降尺度。
func toImageBBox(bb bbox, ss int) bbox {
	if bb.x1 <= bb.x0 || bb.y1 <= bb.y0 {
		return bbox{}
	}
	return bbox{
		x0: int(math.Floor(float64(bb.x0) / float64(ss))),
		y0: int(math.Floor(float64(bb.y0) / float64(ss))),
		x1: int(math.Ceil(float64(bb.x1) / float64(ss))),
		y1: int(math.Ceil(float64(bb.y1) / float64(ss))),
	}
}

// userRect 用户坐标包围盒（渐变 objectBoundingBox 使用）。
type userRect struct{ x, y, w, h float64 }

func userBBox(src [][]Point) userRect {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, c := range src {
		for _, p := range c {
			if p.X < minX {
				minX = p.X
			}
			if p.Y < minY {
				minY = p.Y
			}
			if p.X > maxX {
				maxX = p.X
			}
			if p.Y > maxY {
				maxY = p.Y
			}
		}
	}
	if math.IsInf(minX, 1) {
		return userRect{}
	}
	return userRect{minX, minY, maxX - minX, maxY - minY}
}

// composite 按 mask 覆盖率把 paint 合成到图像（source-over）。
func composite(img *image.RGBA, mask []byte, mw, ss int, bb bbox, paint Paint, opacity float64, inv Matrix, ur userRect) {
	if paint.None || paint.Grad == nil && paint.Col.A == 0 {
		return
	}
	imgW, imgH := img.Bounds().Dx(), img.Bounds().Dy()
	x0, y0 := bb.x0, bb.y0
	x1, y1 := bb.x1, bb.y1
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > imgW {
		x1 = imgW
	}
	if y1 > imgH {
		y1 = imgH
	}
	area := float64(ss * ss)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			cov := 0
			for sy := 0; sy < ss; sy++ {
				row := (y*ss + sy) * mw
				base := row + x*ss
				for sx := 0; sx < ss; sx++ {
					if mask[base+sx] != 0 {
						cov++
					}
				}
			}
			if cov == 0 {
				continue
			}
			col := paint.Col
			if paint.Grad != nil {
				// inv 映射的是超采样设备坐标；像素中心需先升到该坐标系再求用户坐标
				col = gradientAt(paint.Grad, inv, ur,
					(float64(x)+0.5)*float64(ss), (float64(y)+0.5)*float64(ss))
			}
			sa := (float64(cov) / area) * col.A * opacity
			if sa <= 0 {
				continue
			}
			if sa > 1 {
				sa = 1
			}
			i := img.PixOffset(x, y)
			dr := float64(img.Pix[i]) / 255
			dg := float64(img.Pix[i+1]) / 255
			db := float64(img.Pix[i+2]) / 255
			da := float64(img.Pix[i+3]) / 255
			oa := sa + da*(1-sa)
			if oa <= 0 {
				continue
			}
			or := (col.R*sa + dr*da*(1-sa)) / oa
			og := (col.G*sa + dg*da*(1-sa)) / oa
			ob := (col.B*sa + db*da*(1-sa)) / oa
			img.Pix[i] = clampU8(or)
			img.Pix[i+1] = clampU8(og)
			img.Pix[i+2] = clampU8(ob)
			img.Pix[i+3] = clampU8(oa)
		}
	}
}

// gradientAt 计算设备坐标像素处的渐变颜色。
func gradientAt(g *Gradient, inv Matrix, ur userRect, dx, dy float64) Color {
	p := inv.Apply(Point{dx, dy})
	var ax, ay, bx, by float64
	if g.UserSpace {
		ax, ay, bx, by = g.X1, g.Y1, g.X2, g.Y2
	} else {
		ax = ur.x + g.X1*ur.w
		ay = ur.y + g.Y1*ur.h
		bx = ur.x + g.X2*ur.w
		by = ur.y + g.Y2*ur.h
	}
	vx, vy := bx-ax, by-ay
	d2 := vx*vx + vy*vy
	if d2 == 0 {
		return g.At(0)
	}
	t := ((p.X-ax)*vx + (p.Y-ay)*vy) / d2
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return g.At(t)
}

func clampU8(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(v*255 + 0.5)
}

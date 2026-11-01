package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// 数学常量与函数别名（svg.go 中几何辅助使用）。
const math_Pi = math.Pi

var (
	cos = math.Cos
	sin = math.Sin
)

// PathParser 把 SVG path 的 d 属性解析为子路径点序列（曲线按弦长展平为折线）。
type PathParser struct {
	data string
	pos  int

	cur     Point   // 当前点
	start   Point   // 当前子路径起点
	lastC   []Point // 上一条三次贝塞尔的控制点（用于 S）
	lastQ   Point   // 上一条二次贝塞尔的控制点（用于 T）
	lastCmd string

	subs   [][]Point
	curSub []Point
}

// ParsePath 解析 d 属性，返回子路径列表（闭合子路径首尾点相同）。
func ParsePath(d string) ([][]Point, error) {
	p := &PathParser{data: d}
	if err := p.parse(); err != nil {
		return nil, err
	}
	return p.subs, nil
}

func (p *PathParser) parse() error {
	for p.skipSpace() {
		cmd, err := p.readCmd()
		if err != nil {
			return err
		}
		if err := p.doCmd(cmd); err != nil {
			return err
		}
	}
	p.closeSub()
	return nil
}

func (p *PathParser) skipSpace() bool {
	for p.pos < len(p.data) {
		r := rune(p.data[p.pos])
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == ',' {
			p.pos++
			continue
		}
		return true
	}
	return false
}

// readCmd 读取命令字母；后续重复参数沿用上一条命令（M 之后视作 L）。
func (p *PathParser) readCmd() (byte, error) {
	if p.pos >= len(p.data) {
		return 0, fmt.Errorf("d 属性意外结束")
	}
	c := p.data[p.pos]
	if isCmdLetter(c) {
		p.pos++
		return c, nil
	}
	if p.lastCmd == "" {
		return 0, fmt.Errorf("d 属性缺少命令字母: %q", string(c))
	}
	switch p.lastCmd {
	case "M":
		return 'L', nil
	case "m":
		return 'l', nil
	case "Z", "z":
		return 0, fmt.Errorf("闭合命令后不可跟参数")
	default:
		return p.lastCmd[0], nil
	}
}

func isCmdLetter(c byte) bool {
	switch c {
	case 'M', 'm', 'L', 'l', 'H', 'h', 'V', 'v', 'C', 'c', 'S', 's', 'Q', 'q', 'T', 't', 'A', 'a', 'Z', 'z':
		return true
	}
	return false
}

// readNum 读取一个数字（支持科学计数法与负号）。
func (p *PathParser) readNum() (float64, error) {
	p.skipSpace()
	start := p.pos
	if p.pos < len(p.data) && (p.data[p.pos] == '-' || p.data[p.pos] == '+') {
		p.pos++
	}
	seenDigit := false
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		if (c >= '0' && c <= '9') || c == '.' {
			if c != '.' {
				seenDigit = true
			}
			p.pos++
			continue
		}
		if !seenDigit {
			break
		}
		// 指数
		if c == 'e' || c == 'E' {
			p.pos++
			if p.pos < len(p.data) && (p.data[p.pos] == '-' || p.data[p.pos] == '+') {
				p.pos++
			}
			seenDigit = false
			continue
		}
		break
	}
	if start == p.pos {
		return 0, fmt.Errorf("位置 %d 处缺少数字", p.pos)
	}
	v, err := strconv.ParseFloat(p.data[start:p.pos], 64)
	if err != nil {
		return 0, fmt.Errorf("位置 %d 处数字无效: %w", start, err)
	}
	return v, nil
}

func (p *PathParser) doCmd(cmd byte) error {
	rel := unicode.IsLower(rune(cmd))
	upper := byte(unicode.ToUpper(rune(cmd)))
	switch upper {
	case 'Z':
		p.closeSub()
		p.lastCmd = string(cmd)
		return nil
	case 'M', 'L', 'T':
		x, err := p.readNum()
		if err != nil {
			return err
		}
		y, err := p.readNum()
		if err != nil {
			return err
		}
		pt := Point{x, y}
		if rel {
			pt = Point{p.cur.X + x, p.cur.Y + y}
		}
		switch upper {
		case 'M':
			p.closeSub()
			p.curSub = []Point{pt}
			p.cur, p.start = pt, pt
			p.lastC, p.lastQ, p.lastCmd = nil, Point{}, string(cmd)
		case 'L':
			p.ensureSub()
			p.curSub = append(p.curSub, pt)
			p.cur = pt
			p.lastC, p.lastQ, p.lastCmd = nil, Point{}, string(cmd)
		case 'T':
			// 平滑二次：控制点为上一条 Q 的镜像；无前驱时退化为直线
			c := p.cur
			if p.lastQ != (Point{}) || p.lastCmd == "Q" || p.lastCmd == "q" || p.lastCmd == "T" || p.lastCmd == "t" {
				c = Point{2*p.cur.X - p.lastQ.X, 2*p.cur.Y - p.lastQ.Y}
			}
			p.ensureSub()
			p.appendQuad(c, pt)
			p.lastQ = c
			p.cur = pt
			p.lastC = nil
			p.lastCmd = string(cmd)
		}
		return nil
	case 'H':
		x, err := p.readNum()
		if err != nil {
			return err
		}
		if rel {
			x += p.cur.X
		}
		p.ensureSub()
		p.curSub = append(p.curSub, Point{x, p.cur.Y})
		p.cur = Point{x, p.cur.Y}
		p.lastC, p.lastQ, p.lastCmd = nil, Point{}, string(cmd)
		return nil
	case 'V':
		y, err := p.readNum()
		if err != nil {
			return err
		}
		if rel {
			y += p.cur.Y
		}
		p.ensureSub()
		p.curSub = append(p.curSub, Point{p.cur.X, y})
		p.cur = Point{p.cur.X, y}
		p.lastC, p.lastQ, p.lastCmd = nil, Point{}, string(cmd)
		return nil
	case 'C':
		x1, err := p.readNum()
		if err != nil {
			return err
		}
		y1, err := p.readNum()
		if err != nil {
			return err
		}
		x2, err := p.readNum()
		if err != nil {
			return err
		}
		y2, err := p.readNum()
		if err != nil {
			return err
		}
		x, err := p.readNum()
		if err != nil {
			return err
		}
		y, err := p.readNum()
		if err != nil {
			return err
		}
		c1, c2, end := Point{x1, y1}, Point{x2, y2}, Point{x, y}
		if rel {
			c1 = Point{p.cur.X + x1, p.cur.Y + y1}
			c2 = Point{p.cur.X + x2, p.cur.Y + y2}
			end = Point{p.cur.X + x, p.cur.Y + y}
		}
		p.ensureSub()
		p.appendCubic(c1, c2, end)
		p.lastC = []Point{c2}
		p.lastQ, p.cur, p.lastCmd = Point{}, end, string(cmd)
		return nil
	case 'S':
		x2, err := p.readNum()
		if err != nil {
			return err
		}
		y2, err := p.readNum()
		if err != nil {
			return err
		}
		x, err := p.readNum()
		if err != nil {
			return err
		}
		y, err := p.readNum()
		if err != nil {
			return err
		}
		c2, end := Point{x2, y2}, Point{x, y}
		if rel {
			c2 = Point{p.cur.X + x2, p.cur.Y + y2}
			end = Point{p.cur.X + x, p.cur.Y + y}
		}
		c1 := c2
		if len(p.lastC) > 0 && strings.ContainsRune("CcSs", rune(p.lastCmdByte())) {
			c1 = Point{2*p.cur.X - p.lastC[0].X, 2*p.cur.Y - p.lastC[0].Y}
		}
		p.ensureSub()
		p.appendCubic(c1, c2, end)
		p.lastC = []Point{c2}
		p.lastQ, p.cur, p.lastCmd = Point{}, end, string(cmd)
		return nil
	case 'Q':
		x1, err := p.readNum()
		if err != nil {
			return err
		}
		y1, err := p.readNum()
		if err != nil {
			return err
		}
		x, err := p.readNum()
		if err != nil {
			return err
		}
		y, err := p.readNum()
		if err != nil {
			return err
		}
		c, end := Point{x1, y1}, Point{x, y}
		if rel {
			c = Point{p.cur.X + x1, p.cur.Y + y1}
			end = Point{p.cur.X + x, p.cur.Y + y}
		}
		p.ensureSub()
		p.appendQuad(c, end)
		p.lastQ, p.lastC, p.cur, p.lastCmd = c, nil, end, string(cmd)
		return nil
	case 'A':
		rx, err := p.readNum()
		if err != nil {
			return err
		}
		ry, err := p.readNum()
		if err != nil {
			return err
		}
		rot, err := p.readNum()
		if err != nil {
			return err
		}
		fa, err := p.readNum()
		if err != nil {
			return err
		}
		fs, err := p.readNum()
		if err != nil {
			return err
		}
		x, err := p.readNum()
		if err != nil {
			return err
		}
		y, err := p.readNum()
		if err != nil {
			return err
		}
		end := Point{x, y}
		if rel {
			end = Point{p.cur.X + x, p.cur.Y + y}
		}
		p.ensureSub()
		p.appendArc(p.cur, rx, ry, rot, fa != 0, fs != 0, end)
		p.cur = end
		p.lastC, p.lastQ, p.lastCmd = nil, Point{}, string(cmd)
		return nil
	}
	return fmt.Errorf("不支持的 path 命令: %c", cmd)
}

func (p *PathParser) lastCmdByte() rune {
	if p.lastCmd == "" {
		return 0
	}
	return rune(p.lastCmd[0])
}

func (p *PathParser) ensureSub() {
	if p.curSub == nil {
		p.curSub = []Point{p.cur}
	}
}

func (p *PathParser) closeSub() {
	if len(p.curSub) >= 2 {
		p.subs = append(p.subs, p.curSub)
	}
	p.curSub = nil
	if len(p.subs) > 0 {
		p.start = p.subs[len(p.subs)-1][0]
	}
}

// appendCubic 三次贝塞尔按自适应步数展平。
func (p *PathParser) appendCubic(c1, c2, end Point) {
	steps := bezierSteps(p.cur, c1, c2, end)
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		p.curSub = append(p.curSub, cubicAt(p.cur, c1, c2, end, t))
	}
}

func (p *PathParser) appendQuad(c, end Point) {
	steps := bezierSteps(p.cur, c, c, end)
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		p.curSub = append(p.curSub, quadAt(p.cur, c, end, t))
	}
}

func cubicAt(p0, c1, c2, p1 Point, t float64) Point {
	u := 1 - t
	a := u * u * u
	b := 3 * u * u * t
	c := 3 * u * t * t
	d := t * t * t
	return Point{
		X: a*p0.X + b*c1.X + c*c2.X + d*p1.X,
		Y: a*p0.Y + b*c1.Y + c*c2.Y + d*p1.Y,
	}
}

func quadAt(p0, c, p1 Point, t float64) Point {
	u := 1 - t
	a := u * u
	b := 2 * u * t
	d := t * t
	return Point{
		X: a*p0.X + b*c.X + d*p1.X,
		Y: a*p0.Y + b*c.Y + d*p1.Y,
	}
}

// bezierSteps 按控制多边形周长估算展平步数（8~64）。
func bezierSteps(p0, c1, c2, p1 Point) int {
	per := dist(p0, c1) + dist(c1, c2) + dist(c2, p1)
	chord := dist(p0, p1)
	n := int((per+chord)/4) + 6
	if n < 8 {
		n = 8
	}
	if n > 64 {
		n = 64
	}
	return n
}

func dist(a, b Point) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// appendArc 椭圆弧展平（SVG endpoint → center 参数化）。
func (p *PathParser) appendArc(from Point, rx, ry, rotDeg float64, largeArc, sweep bool, to Point) {
	if rx == 0 || ry == 0 || from == to {
		p.curSub = append(p.curSub, to)
		return
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	phi := rotDeg * math.Pi / 180
	cp, sp := math.Cos(phi), math.Sin(phi)

	dx2, dy2 := (from.X-to.X)/2, (from.Y-to.Y)/2
	x1p := cp*dx2 + sp*dy2
	y1p := -sp*dx2 + cp*dy2

	lambda := (x1p*x1p)/(rx*rx) + (y1p*y1p)/(ry*ry)
	if lambda > 1 {
		s := math.Sqrt(lambda)
		rx *= s
		ry *= s
	}
	sign := 1.0
	if largeArc == sweep {
		sign = -1
	}
	num := rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p
	den := rx*rx*y1p*y1p + ry*ry*x1p*x1p
	coef := 0.0
	if den > 0 {
		coef = sign * math.Sqrt(math.Max(0, num/den))
	}
	cxp := coef * (rx * y1p / ry)
	cyp := coef * (-ry * x1p / rx)
	cx := cp*cxp - sp*cyp + (from.X+to.X)/2
	cy := sp*cxp + cp*cyp + (from.Y+to.Y)/2

	ux, uy := (x1p-cxp)/rx, (y1p-cyp)/ry
	vx, vy := (-x1p-cxp)/rx, (-y1p-cyp)/ry
	theta1 := angleBetween(1, 0, ux, uy)
	dtheta := angleBetween(ux, uy, vx, vy)
	if !sweep && dtheta > 0 {
		dtheta -= 2 * math.Pi
	}
	if sweep && dtheta < 0 {
		dtheta += 2 * math.Pi
	}
	steps := int(math.Ceil(math.Abs(dtheta)/(math.Pi/18))) + 4
	if steps < 8 {
		steps = 8
	}
	if steps > 128 {
		steps = 128
	}
	for i := 1; i <= steps; i++ {
		th := theta1 + dtheta*float64(i)/float64(steps)
		px := rx * math.Cos(th)
		py := ry * math.Sin(th)
		p.curSub = append(p.curSub, Point{
			X: cp*px - sp*py + cx,
			Y: sp*px + cp*py + cy,
		})
	}
}

// angleBetween 返回向量 u 到 v 的有向夹角（-π..π）。
func angleBetween(ux, uy, vx, vy float64) float64 {
	nu := math.Sqrt(ux*ux + uy*uy)
	nv := math.Sqrt(vx*vx + vy*vy)
	if nu == 0 || nv == 0 {
		return 0
	}
	c := (ux*vx + uy*vy) / (nu * nv)
	if c > 1 {
		c = 1
	}
	if c < -1 {
		c = -1
	}
	a := math.Acos(c)
	if ux*vy-uy*vx < 0 {
		a = -a
	}
	return a
}

package orch

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// The orch door cut. 32 frames, 96×48, 10fps. Four colours, one paper strip,
// no tower, no title, no HUD. Type owns the credit; we only draw the picture.
const (
	cutW    = 96
	cutH    = 48
	cutN    = 32
	cutFPS  = 10
	cutMS   = 100
	longMS  = cutN * cutMS // 3200
	shortMS = 8 * cutMS    // last still-energy beat
	frameMS = cutMS
)

const (
	paper  = "#F3F2F8"
	ink    = "#191923"
	mute   = "#8a8796"
	accent = "#5B4FCF"
)

const (
	pxPaper byte = iota
	pxInk
	pxMute
	pxAccent
)

const (
	stripY     = 40
	orchH      = 26
	orchW      = 20
	gruntH     = 8
	gruntW     = 6
	orchStandY = stripY - orchH
	orchSitY   = orchStandY + 6
	gruntY     = stripY - gruntH
)

type orchKind int

const (
	orchWalkEven orchKind = iota
	orchWalkOdd
	orchPlant
	orchWobble
	orchSit
)

type gruntPos struct {
	x, y int
	duck bool
	tool bool
}

type canvas struct {
	w, h int
	pix  []byte
}

func newCanvas(w, h int) canvas {
	return canvas{w: w, h: h, pix: make([]byte, w*h)}
}

func (c canvas) at(x, y int) byte {
	if x < 0 || y < 0 || x >= c.w || y >= c.h {
		return pxPaper
	}
	return c.pix[y*c.w+x]
}

func (c canvas) set(x, y int, p byte) {
	if x < 0 || y < 0 || x >= c.w || y >= c.h {
		return
	}
	c.pix[y*c.w+x] = p
}

func (c canvas) rect(x, y, w, h int, p byte) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			c.set(xx, yy, p)
		}
	}
}

func (c canvas) dot(x, y int, p byte) { c.set(x, y, p) }

// scene places everyone for one frame of the 32-frame cut (0-based).
func scene(f int) (ox, oy int, kind orchKind, gs [4]gruntPos, dust [][2]int, showCredit bool) {
	if f < 0 {
		f = 0
	}
	if f > 31 {
		f = 31
	}
	showCredit = f >= 26
	hop := f%2 == 0
	gy := gruntY
	if hop {
		gy = gruntY - 2
	}

	switch {
	case f <= 3:
		// idle shop: pack of 4, empty left third, tools tick
		ox, oy, kind = -28, orchStandY, orchWalkEven
		xs := [4]int{40, 52, 64, 76}
		for i, x := range xs {
			gs[i] = gruntPos{x: x + (f % 2), y: gy, tool: f%2 == 1}
		}
	case f <= 7:
		// orch barrels in; one foot overshoots; grunts freeze on 7
		step := f - 4
		ox = -10 + step*7
		oy = orchStandY
		kind = orchWalkEven
		if step%2 == 1 {
			kind = orchWalkOdd
		}
		if f == 6 {
			kind = orchPlant // foot overshoots
			ox += 2
		}
		if f == 7 {
			kind = orchWalkOdd
			for i, x := range [4]int{42, 54, 66, 78} {
				gs[i] = gruntPos{x: x, y: gruntY}
			}
			break
		}
		for i, x := range [4]int{42, 54, 66, 78} {
			gs[i] = gruntPos{x: x, y: gy, tool: hop}
		}
	case f <= 13:
		// chase: 4px/frame, lurch. Grunts scatter. Dust. No catch.
		step := f - 8
		ox = 16 + step*4
		oy = orchStandY
		kind = orchWalkEven
		if step%2 == 1 {
			kind = orchWalkOdd
		}
		gs[0] = gruntPos{x: 50 + step, y: gruntY - 8 - step}      // up-hop off strip
		gs[1] = gruntPos{x: 58 - step*3, y: gy}                   // reverse
		gs[2] = gruntPos{x: 70 + step, y: gruntY + 3, duck: true} // duck under his step
		gs[3] = gruntPos{x: 80 + step*2, y: gy - 1}               // peel right
		dust = [][2]int{
			{ox + 2, stripY - 1},
			{ox + 3, stripY},
			{ox + orchW - 4, stripY},
			{ox + orchW - 3, stripY - 1},
		}
	case f <= 17:
		// almost-trip: plant, wobble 2 frames. Grunts FREEZE, peek from right.
		ox, oy, kind = 44, orchStandY, orchPlant
		if f == 15 {
			kind = orchWobble
			ox++
		}
		if f == 16 {
			kind = orchWobble
			ox--
		}
		if f == 17 {
			kind = orchPlant
		}
		for i, x := range [4]int{86, 88, 90, 92} {
			gs[i] = gruntPos{x: x, y: gruntY}
		}
	case f <= 23:
		// sits (body -6px), facing them. Hands on knees. Grunts creep back.
		t := f - 18
		ox = 44 - t*4
		if ox < 16 {
			ox = 16
		}
		oy = orchSitY
		kind = orchSit
		from := [4]int{86, 88, 90, 92}
		to := [4]int{46, 56, 66, 76}
		for i := range gs {
			x := from[i] + (to[i]-from[i])*t/5
			gs[i] = gruntPos{x: x, y: gruntY}
		}
	default:
		// last still energy: one ogre watching small workers hop again
		ox, oy, kind = 16, orchSitY, orchSit
		for i, x := range [4]int{46, 56, 66, 76} {
			gs[i] = gruntPos{x: x + (f % 2), y: gy, tool: hop}
		}
	}
	return
}

func paintCut(frame, w, h int) []byte {
	if w < 8 {
		w = cutW
	}
	if h < 8 {
		h = cutH
	}
	c := newCanvas(w, h)
	ox, oy, kind, gs, dust, _ := scene(frame)

	// Empty field is paper (the canvas zero). Ground = one paper strip, read
	// as a strip because a mute edge sits under their feet — not a room.
	edge := muteLine(h)
	c.rect(0, edge, w, 1, pxMute)
	// the strip itself is paper, so it is the field; the mute line is the lip

	sx := scaleX(w)
	sy := scaleY(h)
	drawOrch(c, sx(ox), sy(oy), kind)
	for _, g := range gs {
		drawGrunt(c, sx(g.x), sy(g.y), g.duck, g.tool)
	}
	for _, d := range dust {
		c.dot(sx(d[0]), sy(d[1]), pxMute)
		c.dot(sx(d[0])+1, sy(d[1]), pxMute)
	}
	return c.pix
}

func muteLine(h int) int {
	if h <= 32 {
		return 26
	}
	return stripY
}

func scaleX(w int) func(int) int {
	if w >= cutW {
		return func(x int) int { return x }
	}
	return func(x int) int { return x * w / cutW }
}

func scaleY(h int) func(int) int {
	if h >= cutH {
		return func(y int) int { return y }
	}
	return func(y int) int { return y * h / cutH }
}

func drawOrch(c canvas, x, y int, kind orchKind) {
	lean := 0
	footL, footR := 0, 0
	switch kind {
	case orchWalkEven:
		lean = -1
		footL, footR = -1, 2 // right foot overshoots on the even lurch
	case orchWalkOdd:
		lean = 1
		footL, footR = 2, -1
	case orchPlant:
		footL, footR = 0, 0
	case orchWobble:
		lean = 1
		footL, footR = -1, 1
	case orchSit:
		drawOrchSit(c, x, y)
		return
	}

	l := lean
	// head — too big for him, brow heavy, one tusk
	c.rect(x+5+l, y+0, 10, 1, pxInk)
	c.rect(x+4+l, y+1, 12, 6, pxInk)
	c.rect(x+3+l, y+4, 14, 3, pxInk) // jaw wider than the skull
	c.rect(x+6+l, y+2, 8, 1, pxInk)  // brow
	c.rect(x+6+l, y+3, 2, 2, pxPaper)
	c.rect(x+11+l, y+3, 2, 2, pxPaper)
	c.dot(x+12+l, y+3, pxAccent) // ONE accent pixel: the right eye
	// one tusk, lower left — no crown, no club
	c.dot(x+5+l, y+7, pxInk)
	c.dot(x+4+l, y+8, pxMute)
	c.dot(x+4+l, y+9, pxMute)
	// hunched shoulders, pear body
	c.rect(x+6+l, y+8, 8, 2, pxInk)
	c.rect(x+2+l, y+10, 16, 3, pxInk)
	c.rect(x+3+l, y+13, 14, 5, pxInk)
	// arms — clumsy, hanging past the hips
	c.rect(x+0+l, y+11, 3, 7, pxInk)
	c.rect(x+17+l, y+11, 3, 7, pxInk)
	c.rect(x+0+l, y+17, 2, 2, pxInk)
	c.rect(x+18+l, y+17, 2, 2, pxInk)
	// legs
	c.rect(x+4+l, y+18, 5, 6, pxInk)
	c.rect(x+11+l, y+18, 5, 6, pxInk)
	// wide feet
	c.rect(x+1+l+footL, y+24, 8, 2, pxInk)
	c.rect(x+11+l+footR, y+24, 8, 2, pxInk)
}

func drawOrchSit(c canvas, x, y int) {
	// same head, body dropped, hands on knees, facing the workers
	c.rect(x+5, y+0, 10, 1, pxInk)
	c.rect(x+4, y+1, 12, 6, pxInk)
	c.rect(x+3, y+4, 14, 3, pxInk)
	c.rect(x+6, y+2, 8, 1, pxInk)
	c.rect(x+6, y+3, 2, 2, pxPaper)
	c.rect(x+11, y+3, 2, 2, pxPaper)
	c.dot(x+12, y+3, pxAccent)
	c.dot(x+5, y+7, pxInk)
	c.dot(x+4, y+8, pxMute)
	c.dot(x+4, y+9, pxMute)
	c.rect(x+6, y+8, 8, 2, pxInk)
	c.rect(x+3, y+10, 14, 6, pxInk)
	// hands on knees
	c.rect(x+1, y+13, 3, 4, pxInk)
	c.rect(x+16, y+13, 3, 4, pxInk)
	c.rect(x+0, y+16, 4, 2, pxInk)
	c.rect(x+16, y+16, 4, 2, pxInk)
	// folded legs, wide seat
	c.rect(x+2, y+16, 16, 6, pxInk)
	c.rect(x+1, y+21, 8, 3, pxInk)
	c.rect(x+11, y+21, 8, 3, pxInk)
}

func drawGrunt(c canvas, x, y int, duck, tool bool) {
	if duck {
		c.rect(x+1, y+3, 4, 3, pxInk)
		c.rect(x+1, y+4, 1, 1, pxPaper)
		c.rect(x+3, y+4, 1, 1, pxPaper)
		c.rect(x+0, y+6, 2, 2, pxInk)
		c.rect(x+3, y+6, 2, 2, pxInk)
		if tool {
			c.dot(x+5, y+5, pxMute)
		}
		return
	}
	// 6×8, same face family: two-eye gremlin, hop is a y shift by the caller
	c.rect(x+1, y+0, 4, 1, pxInk)
	c.rect(x+0, y+1, 6, 3, pxInk)
	c.dot(x+1, y+2, pxPaper)
	c.dot(x+4, y+2, pxPaper)
	c.rect(x+1, y+4, 4, 2, pxInk)
	c.dot(x+0, y+6, pxInk)
	c.dot(x+2, y+6, pxInk)
	c.dot(x+3, y+7, pxInk)
	c.dot(x+5, y+6, pxInk)
	if tool {
		c.dot(x+6, y+3, pxMute) // 1px tick
	}
}

func cutGeom(termW, termH int) (w, h, scale int) {
	w, h, scale = cutW, cutH, 1
	if termW >= cutW*3 && termH >= (cutH/2)*3+2 {
		return cutW, cutH, 3
	}
	if termW >= cutW*2 && termH >= (cutH/2)*2+2 {
		return cutW, cutH, 2
	}
	if termW > 0 && termW < cutW {
		return 64, 32, 1
	}
	return w, h, scale
}

var cutPalette = []string{paper, ink, mute, accent}

// RenderCut draws one frame of the orch door. Credit is real type, mute,
// bottom-right, frames 27–32 only. No invented letterform.
func RenderCut(frame, termW, termH int) string {
	w, h, scale := cutGeom(termW, termH)
	pix := paintCut(frame, w, h)
	_, _, _, _, _, credit := scene(frame)
	return renderPixels(pix, w, h, scale, credit)
}

// LastStill is frame 32: he sits, they work, `by manymoats` is already there.
func LastStill(termW, termH int) string {
	return RenderCut(31, termW, termH)
}

func renderPixels(pix []byte, w, h, scale int, credit bool) string {
	if scale < 1 {
		scale = 1
	}
	if h%2 != 0 {
		h--
	}
	var b strings.Builder
	style := func(top, bot byte) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(cutPalette[top])).
			Background(lipgloss.Color(cutPalette[bot]))
	}
	for y := 0; y < h; y += 2 {
		for range scale {
			for x := 0; x < w; x++ {
				top := pix[y*w+x]
				bot := pxPaper
				if y+1 < h {
					bot = pix[(y+1)*w+x]
				}
				cell := style(top, bot).Render("▀")
				for range scale {
					b.WriteString(cell)
				}
			}
			b.WriteByte('\n')
		}
	}
	if credit {
		// Type owns letters. Mute, bottom-right, lowercase.
		pad := w*scale - len("by manymoats")
		if pad < 0 {
			pad = 0
		}
		b.WriteString(strings.Repeat(" ", pad))
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(mute)).Render("by manymoats"))
		b.WriteByte('\n')
	}
	return b.String()
}

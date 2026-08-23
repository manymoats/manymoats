package agent

import (
	"strings"
	"sync"

	"github.com/mattn/go-runewidth"
)

var (
	mu       sync.RWMutex
	ambigDbl bool
)

func SetAmbiguousWide(wide bool) {
	mu.Lock()
	ambigDbl = wide
	runewidth.DefaultCondition.EastAsianWidth = wide
	mu.Unlock()
}

func AmbiguousWide() bool {
	mu.RLock()
	defer mu.RUnlock()
	return ambigDbl
}

// Render returns the single-cell mark. The drawn 3x3 form looked right in an HTML
// render where font-size could scale it, but a terminal cannot scale a glyph — so
// there it only ate three rows per agent and made the board sparse. Big() keeps
// the drawn form for the splash and the animation, where space is the point.
// Render is the board icon: the glyph itself, nothing around it. Frames were
// decoration, not size.
func (m Mark) Render() (top, mid, bot string) {
	return "", m.safe(m.Glyph), ""
}

// Tiny is the one-cell form, for legends and the minimal strip.
func (m Mark) Tiny() string {
	return m.safe(m.Glyph)
}

func (m Mark) Big() (top, mid, bot string) {
	mid = m.safe(m.Mid)
	if mid == m.ASCII {
		return "", m.ASCII, ""
	}
	return m.Top, mid, m.Bot
}

// safe never emits a glyph that will take the wrong number of cells. LANG
// does not flip the whole board to ASCII — only this glyph is measured.
// A blank ASCII leftover is worse than a plain middle-dot, so we never
// return a space and call it an icon.
func (m Mark) safe(g string) string {
	if g != "" && GlyphFits(g) {
		return g
	}
	if strings.TrimSpace(m.ASCII) != "" {
		return m.ASCII
	}
	return "·"
}

func Width(s string) int { return runewidth.StringWidth(s) }

func GlyphFits(s string) bool {
	if s == "" {
		return false
	}
	return runewidth.StringWidth(s) == len([]rune(s))
}

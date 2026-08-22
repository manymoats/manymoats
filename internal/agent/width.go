package agent

import (
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
	if AmbiguousWide() {
		return "", m.ASCII, ""
	}
	return "", m.Glyph, ""
}

// Tiny is the one-cell form, for legends and the minimal strip.
func (m Mark) Tiny() string {
	if AmbiguousWide() {
		return m.ASCII
	}
	return m.Glyph
}

func (m Mark) Big() (top, mid, bot string) {
	if AmbiguousWide() {
		return "", m.ASCII, ""
	}
	return m.Top, m.Mid, m.Bot
}

func Width(s string) int { return runewidth.StringWidth(s) }

func GlyphFits(s string) bool { return runewidth.StringWidth(s) == len([]rune(s)) }

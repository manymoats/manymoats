package agent

import (
	"fmt"
	"strings"
)

// Tower — the lookout. You watch the fleet from it; you never steer from it.
var Tower = []string{
	"  ▄▄▄  ",
	" ▟███▙ ",
	" █ ◉ █ ",
	" ▜███▛ ",
	"  ███  ",
	" ▄███▄ ",
	"▟█████▙",
}

var TowerSmall = []string{
	" ▄▄ ",
	"▟◉█▙",
	" ██ ",
	"▟██▙",
}

// Silver is a brushed-metal ramp. A terminal cannot render a gradient inside one
// cell, so the gradient runs ACROSS cells — which is how metal reads in text.
var Silver = []string{
	"#6E7681", "#8B949E", "#A8B3BD", "#C9D1D9", "#E6EDF3", "#C9D1D9", "#A8B3BD", "#8B949E",
}

func SilverAt(i int) string { return Silver[((i%len(Silver))+len(Silver))%len(Silver)] }

// Metal renders s with the silver ramp swept across it. phase animates the sweep.
func Metal(s string, phase int, paint func(hex, text string) string) string {
	var b strings.Builder
	for i, r := range []rune(s) {
		if r == ' ' {
			b.WriteRune(r)
			continue
		}
		b.WriteString(paint(SilverAt(i+phase), string(r)))
	}
	return b.String()
}

// DotGround is the field the board sits on — the house's own ground language,
// drawn at the lowest intensity a terminal has.
func DotGround(w, h int, spacing int) []string {
	if spacing < 2 {
		spacing = 4
	}
	rows := make([]string, 0, h)
	for y := 0; y < h; y++ {
		var b strings.Builder
		for x := 0; x < w; x++ {
			if y%spacing == 0 && x%spacing == 0 {
				b.WriteRune('·')
			} else {
				b.WriteRune(' ')
			}
		}
		rows = append(rows, b.String())
	}
	return rows
}

// SetTerminalBackground returns the OSC 11 sequence that asks the terminal to
// change its own background. Supported by iTerm2, Ghostty, Kitty, WezTerm, xterm.
// Terminal.app ignores it. Always pair with ResetTerminalBackground on exit —
// leaving someone's terminal recoloured after quitting is rude.
func SetTerminalBackground(hex string) string { return fmt.Sprintf("\x1b]11;%s\x07", hex) }

func ResetTerminalBackground() string { return "\x1b]111\x07" }

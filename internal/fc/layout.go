package fc

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/manymoats/manymoats/internal/credits"
)

// The frame is 80 columns because that is the width the render was measured at
// and the width a terminal is still allowed to be.
const frameW = 80

// The row grammar the whole tool inherits, in cells. Measured off the approved
// render, not guessed:
//
//	│ ○ name(28)      what's left(11)  when it dies(18) how old(14) │
//
// gutter + marker + gap + name + left + gap2 + dies + gap3 + age + tail == 78,
// and 78 + the two frame walls == 80. markerW+nameW is held at 29 so the plain
// word markers ("live" / "part" / "old") cost the name column and nothing else.
const (
	leftW = 11
	diesW = 18
	ageW  = 14
	nameW = 29
)

// Width is measured with lipgloss, never runewidth on a styled string: the
// latter counts ANSI escape bytes as visible columns, which stays invisible
// until a double-width rune arrives and then shears every column by one.
func w(s string) int { return lipgloss.Width(s) }

func pad(s string, n int) string {
	if d := n - w(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return clip(s, n)
}

func padL(s string, n int) string {
	if d := n - w(s); d > 0 {
		return strings.Repeat(" ", d) + s
	}
	return clip(s, n)
}

// clip shortens without ever spending a character to say a character is
// missing. It cuts at a separator when there is one, and hard-truncates when
// there is not. There is no ellipsis anywhere in this binary.
func clip(s string, n int) string {
	if w(s) <= n {
		return s
	}
	for _, sep := range []string{" · ", " — ", " (", ", "} {
		if i := strings.Index(s, sep); i > 0 && w(s[:i]) <= n {
			return pad(s[:i], n)
		}
	}
	r := []rune(s)
	for len(r) > 0 && w(string(r)) > n {
		r = r[:len(r)-1]
	}
	return pad(strings.TrimRight(string(r), " "), n)
}

func rule(n int) string  { return strings.Repeat("─", n) }
func heavy(n int) string { return strings.Repeat("═", n) }

// A frame in ascii mode is not a downgrade for looks. Box-drawing glyphs are
// East-Asian Ambiguous exactly like the markers are, so a terminal that gives
// them two cells shears every line of every frame. Swapping the markers alone
// would have been half a fix.
type frame struct {
	width int
	ascii bool
}

func (f frame) rule(n int) string {
	if f.ascii {
		return strings.Repeat("-", n)
	}
	return rule(n)
}

func (f frame) corners() (tl, tr, bl, br, tee1, tee2, wall string) {
	if f.ascii {
		return "+", "+", "+", "+", "+", "+", "|"
	}
	return "┌", "┐", "└", "┘", "├", "┤", "│"
}

func (f frame) top() string {
	tl, tr, _, _, _, _, _ := f.corners()
	return tl + f.rule(f.width-2) + tr
}

func (f frame) mid() string {
	_, _, _, _, t1, t2, _ := f.corners()
	return t1 + f.rule(f.width-2) + t2
}

func (f frame) bot() string {
	_, _, bl, br, _, _, _ := f.corners()
	return bl + f.rule(f.width-2) + br
}

// line takes content that already carries its own left inset, so the caller
// controls the column and the frame only guarantees the width.
func (f frame) line(inner string) string {
	_, _, _, _, _, _, wall := f.corners()
	return wall + pad(inner, f.width-2) + wall
}

func (f frame) heavyRule(n int) string {
	if f.ascii {
		return strings.Repeat("=", n)
	}
	return heavy(n)
}

func (f frame) heavyTop() string {
	if f.ascii {
		return "#" + f.heavyRule(f.width-2) + "#"
	}
	return "╔" + heavy(f.width-2) + "╗"
}

func (f frame) heavyBot() string {
	if f.ascii {
		return "#" + f.heavyRule(f.width-2) + "#"
	}
	return "╚" + heavy(f.width-2) + "╝"
}

func (f frame) heavyLine(inner string) string {
	if f.ascii {
		return "#" + pad(inner, f.width-2) + "#"
	}
	return "║" + pad(inner, f.width-2) + "║"
}

// marker carries the honesty tier. Colour is never the only carrier: the glyph
// says it, and where the glyph cannot be trusted to be one cell wide the word
// says it instead.
type marks struct{ plain bool }

func (m marks) of(s credits.Method) string {
	if !m.plain {
		return s.Marker()
	}
	switch s {
	case credits.Live:
		return "live"
	case credits.Derived:
		return "part"
	default:
		return "old"
	}
}

func (m marks) width() int {
	if m.plain {
		return 4
	}
	return 1
}

func (m marks) nameW() int { return nameW - m.width() }

// paint is applied AFTER the cell is padded, so colour can never change a
// column. Colour is identity here — which provider a line belongs to — and it
// is never the only carrier of anything.
type paint func(string) string

func hue(colour string, plain bool) paint {
	if plain || colour == "" {
		return func(s string) string { return s }
	}
	st := lipgloss.NewStyle().Foreground(lipgloss.Color(colour))
	return func(s string) string { return st.Render(s) }
}

// row is the one place the credit line is assembled. Every surface that prints
// a credit goes through here, so a column can only drift in one place.
func (m marks) row(f frame, mark, name string, ink paint, left, dies, age string) string {
	if ink == nil {
		ink = func(s string) string { return s }
	}
	inner := " " + pad(mark, m.width()) + " " +
		ink(pad(name, m.nameW())) +
		padL(left, leftW) + "  " +
		pad(dies, diesW) + " " +
		pad(age, ageW) + " "
	return f.line(inner)
}

func (m marks) headRow(f frame) string {
	inner := " " + strings.Repeat(" ", m.width()) + " " +
		pad("credit", m.nameW()) +
		padL("what's left", leftW) + "  " +
		pad("when it dies", diesW) + " " +
		pad("how old", ageW) + " "
	return f.line(inner)
}

// detailW is the one source of truth for how much room a detail line has:
// 5 spaces of inset + the text + 1 space of tail must come to width-2. Having
// the wrapper and the printer each carry their own idea of this is what let a
// sentence get silently cut at its em dash.
func (f frame) detailW() int { return f.width - 8 }

// detail is the plain-words line under a credit: what is spending it, what it
// would take to use it, why nobody can check it.
func (f frame) detail(s string) string {
	return f.line("     " + pad(s, f.detailW()) + " ")
}

func (f frame) prose(s string) string { return f.line("  " + s) }

// fitLines leaves a line that already fits exactly as it was written, because
// wrap collapses runs of spaces and the row grammar uses them deliberately.
func fitLines(s string, n int) []string {
	if w(s) <= n {
		return []string{s}
	}
	return wrap(s, n)
}

// wrap breaks at spaces only. Nothing is hyphenated and nothing is elided.
func wrap(s string, n int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var out []string
	cur := words[0]
	for _, x := range words[1:] {
		if w(cur)+1+w(x) <= n {
			cur += " " + x
			continue
		}
		out = append(out, cur)
		cur = x
	}
	return append(out, cur)
}

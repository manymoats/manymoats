package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/manymoats/manymoats/internal/agent"
)

const width = 48

type mark struct {
	col   int
	glyph string
	hex   string
}

var found = []mark{{10, "✳", "#D97757"}, {20, "✳", "#D97757"}, {30, "⬡", "#A472F0"}, {40, "◱", "#C9D1D9"}}

func paint(hex, s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render(s)
}

func groundRow(lead int) string {
	var b strings.Builder
	for x := 0; x < width; x++ {
		if i := agent.BeamWedge(x, lead); i > 0 {
			b.WriteString(paint(agent.Ramp(i), "─"))
			continue
		}
		ch := " "
		if x%4 == 0 {
			ch = "·"
		}
		b.WriteString(paint("#232a33", ch))
	}
	return b.String()
}

func markRow(lead int) string {
	cells := make([]string, width)
	for i := range cells {
		cells[i] = " "
	}
	for _, m := range found {
		w := agent.Woken(m.col, lead, 3)
		if w <= 0 {
			continue
		}
		hex := m.hex
		if w < 0.6 {
			hex = "#4c5865"
		}
		cells[m.col] = paint(hex, m.glyph)
	}
	return strings.Join(cells, "")
}

func frame(shown int, t float64, word bool) string {
	lead := agent.BeamAt(t, width)
	var b strings.Builder
	rows := agent.Tower
	for i := 0; i < len(rows); i++ {
		line := "   "
		if len(rows)-i <= shown {
			line += agent.Metal(rows[i], 0, paint)
		} else {
			line += strings.Repeat(" ", len([]rune(rows[i])))
		}
		line += "    "
		if word {
			switch i {
			case 1:
				line += agent.Metal("O R C H", int(t*8), paint)
			case 2:
				line += paint("#4c5865", "the lookout")
			case 4:
				line += paint("#3a4450", "by manymoats")
			}
		}
		b.WriteString(strings.TrimRight(line, " ") + "\n")
	}
	b.WriteString("   " + markRow(lead) + "\n")
	b.WriteString("   " + groundRow(lead) + "\n")
	return b.String()
}

func main() {
	steps := []struct {
		shown int
		t     float64
		word  bool
	}{{2, 0, false}, {5, 0, false}, {7, 0, false}, {7, 0.2, false}, {7, 0.45, false}, {7, 0.7, true}, {7, 1.0, true}}
	only := -1
	if len(os.Args) > 1 {
		fmt.Sscanf(os.Args[1], "%d", &only)
	}
	for i, s := range steps {
		if only >= 0 && i != only {
			continue
		}
		fmt.Printf("\n── beat %d ─────────────────────────────────────\n%s", i+1, frame(s.shown, s.t, s.word))
	}
}

package first

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/manymoats/manymoats/internal/agent"
)

// The house cut. Same timing as the orch desk splash — one engine, two doors.
const (
	longMS  = 4200
	frameMS = 40
	beamW   = 46
)

const (
	hero   = "manymoats"
	credit = "by manymoats"
)

func paint(hex, s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render(s)
}

var (
	ground = lipgloss.NewStyle().Foreground(lipgloss.Color("#232a33"))
	quiet  = lipgloss.NewStyle().Foreground(lipgloss.Color("#3a4450"))
)

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// Last is the final frame of a cut. Reduced motion and anything that is not a
// terminal jump here: the name and the credit still appear, nothing waits.
func Last(main bool) string {
	if main {
		return mainFrame(longMS, longMS, 8)
	}
	return smallFrame(0)
}

func mainFrame(elapsed, total, phase int) string {
	p := 1.0
	if total > 0 {
		p = clamp01(float64(elapsed) / float64(total))
	}

	shown := len(agent.Tower)
	if p < 0.22 {
		shown = int(agent.SilkInOut(clamp01(p/0.22)) * float64(len(agent.Tower)))
	}
	windowLit := p >= 0.22
	wordT := clamp01((p - 0.62) / 0.20)
	bt := clamp01((p - 0.27) / 0.35)

	lead := -99
	if bt > 0 {
		lead = agent.BeamAt(bt, beamW)
	}

	word := ""
	if wordT > 0 {
		n := int(agent.Silk(wordT) * float64(len(hero)))
		if n > len(hero) {
			n = len(hero)
		}
		if n < 1 && wordT > 0 {
			n = 1
		}
		word = hero[:n]
	}

	var b strings.Builder
	b.WriteString("\n")
	for i, row := range agent.Tower {
		line := "   "
		if len(agent.Tower)-i <= shown {
			r := row
			if i == 2 && !windowLit {
				r = strings.ReplaceAll(r, "◉", " ")
			}
			line += agent.Metal(r, phase, paint)
		} else {
			line += strings.Repeat(" ", len([]rune(row)))
		}
		line += "    "
		switch i {
		case 1:
			if word != "" {
				line += agent.Metal(word, phase, paint)
			}
		case 4:
			if p >= 0.62 {
				line += quiet.Render(credit)
			}
		}
		b.WriteString(strings.TrimRight(line, " ") + "\n")
	}

	var gr strings.Builder
	for x := 0; x < beamW; x++ {
		if in := agent.BeamWedge(x, lead); in > 0 {
			gr.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(agent.Ramp(in))).Render("─"))
			continue
		}
		ch := " "
		if x%4 == 0 {
			ch = "·"
		}
		gr.WriteString(ground.Render(ch))
	}
	b.WriteString("   " + gr.String() + "\n")
	return b.String()
}

func smallFrame(phase int) string {
	var b strings.Builder
	b.WriteString("\n")
	for i, row := range agent.TowerSmall {
		line := "  " + agent.Metal(row, phase, paint)
		if i == 1 {
			line += "   " + quiet.Render(credit)
		}
		b.WriteString(strings.TrimRight(line, " ") + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

// play runs the long cut. The caller already decided motion is allowed.
func play(out io.Writer) {
	fmt.Fprint(out, "\x1b[?25l")
	defer fmt.Fprint(out, "\x1b[?25h")
	frame := 0
	for elapsed := 0; elapsed <= longMS; elapsed += frameMS {
		fmt.Fprint(out, "\x1b[H\x1b[J")
		fmt.Fprint(out, mainFrame(elapsed, longMS, frame))
		time.Sleep(frameMS * time.Millisecond)
		frame++
	}
}

func noAnimFlag() bool {
	if os.Getenv("ORCH_NO_ANIM") != "" {
		return true
	}
	for _, a := range os.Args[1:] {
		if a == "--no-anim" {
			return true
		}
	}
	return false
}

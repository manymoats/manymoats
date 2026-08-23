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

// The house cut. Same clock as before — the word arrives late, the credit
// lands with it. The picture is the word. The ogre movie lives on the orch
// door, not here.
const (
	longMS  = 4200
	frameMS = 40
)

const (
	hero   = "manymoats"
	credit = "by manymoats"
)

var (
	ink   = lipgloss.NewStyle().Foreground(lipgloss.Color("#191923"))
	quiet = lipgloss.NewStyle().Foreground(lipgloss.Color("#8a8796"))
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
	return smallFrame()
}

func mainFrame(elapsed, total, _ int) string {
	p := 1.0
	if total > 0 {
		p = clamp01(float64(elapsed) / float64(total))
	}

	wordT := clamp01((p - 0.62) / 0.20)
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
	b.WriteString("\n\n")
	line := "   "
	if word != "" {
		line += ink.Bold(true).Render(word)
	}
	b.WriteString(strings.TrimRight(line, " ") + "\n")
	if p >= 0.62 {
		b.WriteString("   " + quiet.Render(credit) + "\n")
	} else {
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

func smallFrame() string {
	return "\n  " + quiet.Render(credit) + "\n\n"
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

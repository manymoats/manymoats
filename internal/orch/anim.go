package orch

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/manymoats/manymoats/internal/agent"
)

const (
	longMS  = 4200
	shortMS = 700
	frameMS = 40
)

type animFrame time.Time

func animTick() tea.Cmd {
	return tea.Tick(frameMS*time.Millisecond, func(t time.Time) tea.Msg { return animFrame(t) })
}

func seenPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".orch", "seen")
}

func firstRun() bool {
	p := seenPath()
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return os.IsNotExist(err)
}

func markSeen() {
	p := seenPath()
	if p == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	_ = os.WriteFile(p, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o600)
}

// motionOK is the reduced-motion gate. A non-TTY, NO_COLOR, or an explicit flag
// gets the final frame instantly — the full picture, never a degraded one.
func motionOK() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("ORCH_NO_ANIM") != "" {
		return false
	}
	for _, a := range os.Args[1:] {
		if a == "--no-anim" {
			return false
		}
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func (m model) animDone() bool {
	total := shortMS
	if m.animLong {
		total = longMS
	}
	return m.animElapsed >= total
}

func (m model) animProgress() float64 {
	total := float64(shortMS)
	if m.animLong {
		total = float64(longMS)
	}
	return float64(m.animElapsed) / total
}

// beamT maps overall progress onto the sweep's own window: the long cut builds the
// tower first and lights the window before the beam moves at all.
func (m model) beamT() float64 {
	p := m.animProgress()
	if !m.animLong {
		return clamp01((p - 0.15) / 0.5)
	}
	return clamp01((p - 0.27) / 0.35)
}

func (m model) towerRows() int {
	if !m.animLong {
		return len(agent.Tower)
	}
	p := m.animProgress()
	if p >= 0.22 {
		return len(agent.Tower)
	}
	return int(agent.SilkInOut(clamp01(p/0.22)) * float64(len(agent.Tower)))
}

func (m model) windowLit() bool {
	return !m.animLong || m.animProgress() >= 0.22
}

func (m model) wordShown() bool {
	if !m.animLong {
		return m.animProgress() >= 0.5
	}
	return m.animProgress() >= 0.62
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func (m model) animView() string {
	const w = 46
	bt := m.beamT()
	// Before the sweep starts there is no light at all. Drawing the leading edge
	// at column 0 makes it look like the beam is already on and stuck.
	lead := -99
	if bt > 0 {
		lead = agent.BeamAt(bt, w)
	}
	shown := m.towerRows()
	rows := agent.Tower

	var b strings.Builder
	b.WriteString("\n")
	for i := range rows {
		line := "   "
		if len(rows)-i <= shown {
			r := rows[i]
			if i == 2 && !m.windowLit() {
				r = strings.ReplaceAll(r, "◉", " ")
			}
			line += agent.Metal(r, m.frame, metalPaint)
		} else {
			line += strings.Repeat(" ", len([]rune(rows[i])))
		}
		line += "    "
		if m.wordShown() {
			switch i {
			case 1:
				line += agent.Metal("O R C H", m.frame, metalPaint)
			case 2:
				line += dim.Render("the lookout")
			case 4:
				line += ground.Render("by manymoats")
			}
		}
		b.WriteString(strings.TrimRight(line, " ") + "\n")
	}

	// marks wake where the beam has already passed — real agents only
	cells := make([]string, w)
	for i := range cells {
		cells[i] = " "
	}
	working := m.workingAgents()
	for i, a := range working {
		col := 6 + i*11
		if col >= w {
			break
		}
		if wake := agent.Woken(col, lead, 3); wake > 0 {
			_, mid, _ := agent.MarkFor(a.Source).Big()
			hex := agent.MarkFor(a.Source).Color
			if wake < 0.6 {
				hex = "#4c5865"
			}
			g := []rune(mid)
			cells[col] = lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render(string(g[len(g)/2]))
		}
	}
	b.WriteString("   " + strings.Join(cells, "") + "\n")

	var gr strings.Builder
	for x := 0; x < w; x++ {
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
	if m.animLong {
		b.WriteString("\n   " + ground.Render("any key to skip") + "\n")
	}
	return b.String()
}

func (m model) workingAgents() []agent.Agent {
	var out []agent.Agent
	for _, a := range m.agents {
		if a.State == agent.Working || a.State == agent.Asks {
			out = append(out, a)
		}
	}
	return out
}

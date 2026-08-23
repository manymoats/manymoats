package orch

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

// motionOK is the reduced-motion gate. A non-TTY, CI, NO_COLOR, or an explicit
// flag gets the final frame instantly — the full picture, never a hang.
func motionOK() bool {
	if os.Getenv("CI") != "" || os.Getenv("NO_COLOR") != "" || os.Getenv("ORCH_NO_ANIM") != "" {
		return false
	}
	for _, a := range os.Args[1:] {
		if a == "--no-anim" {
			return false
		}
	}
	return stdoutTTY()
}

func stdoutTTY() bool {
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

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// cutIndex maps elapsed time onto the 32-frame sheet. First run plays 01–32.
// A return visit plays the watch beat (25–32) so the door is still the picture
// without making them sit through the chase again.
func (m model) cutIndex() int {
	f := m.animElapsed / frameMS
	if m.animLong {
		if f > 31 {
			return 31
		}
		if f < 0 {
			return 0
		}
		return f
	}
	f += 24
	if f > 31 {
		return 31
	}
	if f < 24 {
		return 24
	}
	return f
}

func (m model) animView() string {
	return RenderCut(m.cutIndex(), m.w, m.h)
}

// printLastStill writes frame 32. A pipe / CI caller should then return
// without starting the board, so nothing waits on a key.
func printLastStill() {
	fmt.Print(LastStill(0, 0))
}

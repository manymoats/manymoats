package launcher

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/manymoats/manymoats/internal/version"
)

// An App is one thing this binary can run. Adding a tool to the house means
// adding a line to Apps — the directory, the dispatcher and the help text all
// read from the same list, so they cannot drift apart.
type App struct {
	Name  string
	What  string
	Ready bool
	Run   func() int
}

var (
	bright = lipgloss.NewStyle().Foreground(lipgloss.Color("#A8B3BD"))
	mid    = lipgloss.NewStyle().Foreground(lipgloss.Color("#5c6673"))
	dim    = lipgloss.NewStyle().Foreground(lipgloss.Color("#3a4450"))
	live   = lipgloss.NewStyle().Foreground(lipgloss.Color("#5FD0C0"))
)

func pad(s string, n int) string {
	if w := lipgloss.Width(s); w < n {
		return s + strings.Repeat(" ", n-w)
	}
	return s
}

// Directory is what `manymoats` prints on its own: what you can run, in one
// screen, with no interface to escape from.
func Directory(apps []App) string {
	nameW := 0
	for _, a := range apps {
		if len(a.Name) > nameW {
			nameW = len(a.Name)
		}
	}
	whatW := 0
	for _, a := range apps {
		if len(a.What) > whatW {
			whatW = len(a.What)
		}
	}

	var b strings.Builder
	b.WriteString("\n  " + bright.Bold(true).Render("MANYMOATS") + "  " + dim.Render(version.V) + "\n\n")
	for _, a := range apps {
		state := dim.Render("soon")
		name := mid.Render(a.Name)
		if a.Ready {
			state = live.Render("ready")
			name = bright.Render(a.Name)
		}
		b.WriteString("  " + pad(name, nameW+3) + mid.Render(pad(a.What, whatW+3)) + state + "\n")
	}
	ready := ""
	for _, a := range apps {
		if a.Ready {
			ready = a.Name
			break
		}
	}
	b.WriteString("\n  " + dim.Render("run one:") + "  " + mid.Render("manymoats "+ready) + "\n\n")
	return b.String()
}

// Unknown keeps a typo from being a dead end.
func Unknown(name string, apps []App) string {
	return fmt.Sprintf("\n  %s\n%s",
		mid.Render("no app called ")+bright.Render(name),
		Directory(apps))
}

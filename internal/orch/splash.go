package orch

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/manymoats/manymoats/internal/agent"
)

func metalPaint(hex, s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render(s)
}

var ground = lipgloss.NewStyle().Foreground(lipgloss.Color("#232a33"))

func (m model) splash() string {
	var b strings.Builder
	dotRow := ground.Render(strings.Repeat("·   ", 7))

	b.WriteString("\n")
	for i, row := range agent.Tower {
		line := "    " + agent.Metal(row, m.frame, metalPaint) + "     "
		switch i {
		case 1:
			line += agent.Metal("O R C H", m.frame, metalPaint)
		case 2:
			line += dim.Render("the lookout")
		case 4:
			line += dim.Render("by manymoats")
		case 6:
			line += dotRow
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")

	seen := map[agent.Source]int{}
	for _, a := range m.agents {
		seen[a.Source]++
	}
	if len(seen) > 0 {
		var parts []string
		srcs := make([]agent.Source, 0, len(seen))
		for src := range seen {
			srcs = append(srcs, src)
		}
		sort.Slice(srcs, func(i, j int) bool { return string(srcs[i]) < string(srcs[j]) })
		for _, src := range srcs {
			n := seen[src]
			_, mid, _ := agent.MarkFor(src).Big()
			parts = append(parts,
				lipgloss.NewStyle().Foreground(lipgloss.Color(agent.MarkFor(src).Color)).Render(mid)+
					dim.Render(fmt.Sprintf(" %d %s", n, agent.Brand(src))))
		}
		b.WriteString("    " + strings.Join(parts, "    ") + "\n\n")
	}

	w, a := 0, 0
	for _, x := range m.agents {
		if x.State == agent.Working {
			w++
		}
		if x.State == agent.Asks {
			a++
		}
	}
	status := fmt.Sprintf("%d working", w)
	if a > 0 {
		status += fmt.Sprintf(" · %d waiting on you", a)
	} else if w == 0 {
		status = "all clear"
	}
	b.WriteString("    " + lipgloss.NewStyle().Bold(true).Render(status) + "\n\n")
	b.WriteString("    " + ground.Render(strings.Repeat("·   ", 12)) + "\n\n")
	b.WriteString("    " + dim.Render("1 marks · 2 instrument · 3 waveform · 4 cards · m minimal") + "\n")
	b.WriteString("    " + dim.Render("n names: model / brand / both        enter to begin · q quit") + "\n")
	return b.String()
}

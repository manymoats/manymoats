package eyes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	dim    = lipgloss.NewStyle().Foreground(lipgloss.Color("#5c6673"))
	quiet  = lipgloss.NewStyle().Foreground(lipgloss.Color("#3a4450"))
	bright = lipgloss.NewStyle().Foreground(lipgloss.Color("#A8B3BD"))
	good   = lipgloss.NewStyle().Foreground(lipgloss.Color("#5FD0C0"))
)

func pad(s string, n int) string {
	if w := Cells(s); w < n {
		return s + strings.Repeat(" ", n-w)
	}
	return s
}

// Render draws the report. The verdict is carried by the WORD, never by colour
// alone — colour is identity in this house and a reader without it must still
// be able to read the answer.
func (r Report) Render() string {
	var b strings.Builder
	b.WriteString("\n  " + bright.Bold(true).Render("EYES") + dim.Render("  ·  "+r.Subject) + "\n\n")

	b.WriteString("  " + dim.Render(pad("claim", 34)+pad("measured", 24)+"verdict") + "\n")
	b.WriteString("  " + quiet.Render(strings.Repeat("─", 66)) + "\n")

	agree, disagree, unknown := 0, 0, 0
	for _, c := range r.Claims {
		switch c.Verdict {
		case Agrees:
			agree++
		case Disagrees:
			disagree++
		default:
			unknown++
		}
		v := dim.Render(string(c.Verdict))
		if c.Verdict == Agrees {
			v = good.Render(string(c.Verdict))
		}
		if c.Verdict == Disagrees {
			v = bright.Bold(true).Render(strings.ToUpper(string(c.Verdict)))
		}
		b.WriteString("  " + pad(bright.Render(c.Said), 34) + pad(dim.Render(c.Measured), 24) + v + "\n")
		if c.Why != "" && c.Verdict != Agrees {
			b.WriteString("  " + quiet.Render("    "+c.Why) + "\n")
		}
	}

	b.WriteString("\n  " + dim.Render(fmt.Sprintf("%d checked · %d agree · %d disagree · %d not measured",
		len(r.Claims), agree, disagree, unknown)) + "\n")

	if len(r.Unmeasured) > 0 {
		b.WriteString("\n  " + bright.Render("what this run could not see") + "\n")
		b.WriteString("  " + quiet.Render(strings.Repeat("─", 66)) + "\n")
		for _, u := range r.Unmeasured {
			b.WriteString("  " + dim.Render(u) + "\n")
		}
		// Said out loud, because an unmeasured dimension read as a pass is the
		// failure this whole tool was built to stop.
		b.WriteString("\n  " + dim.Render(fmt.Sprintf("%d not measured · they are not passes", len(r.Unmeasured))) + "\n")
	}

	b.WriteString("\n  " + quiet.Render("by manymoats") + "\n")
	return b.String()
}

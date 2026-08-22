package agents

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
	if w := lipgloss.Width(s); w < n {
		return s + strings.Repeat(" ", n-w)
	}
	return s
}

func head(title string) string {
	return "\n  " + bright.Bold(true).Render("AGENTS") + dim.Render("  ·  "+title) + "\n\n"
}

func rule() string { return "  " + quiet.Render(strings.Repeat("─", 69)) + "\n" }

// Roster. The compute ruling is ON THIS SURFACE, not in a README — nobody
// should have to wonder whether a free thing is quietly metered, and nobody
// should ever write in asking for a key.
func Roster(as []Agent, installed func(string) bool) string {
	var b strings.Builder
	b.WriteString("\n  " + bright.Bold(true).Render("AGENTS") + "\n\n")
	b.WriteString("  " + dim.Render(pad("name", 9)+pad("what it is for", 38)+pad("size", 8)+"installed") + "\n")
	b.WriteString(rule())
	for _, a := range as {
		yes := dim.Render("no")
		if installed(a.Name) {
			yes = good.Render("yes")
		}
		b.WriteString("  " + pad(bright.Render(a.Name), 9) + pad(dim.Render(a.For), 38) + pad(dim.Render(a.Size), 8) + yes + "\n")
	}
	b.WriteString("\n  " + dim.Render("These run on your machine, on your GPU, at your cost. Nothing here") + "\n")
	b.WriteString("  " + dim.Render("calls anything we pay for, and there is no key to ask us for.") + "\n")
	if len(as) > 0 {
		b.WriteString("\n  " + dim.Render("manymoats agents install "+as[0].Name) + "\n")
	}
	b.WriteString("\n  " + quiet.Render("by manymoats") + "\n")
	return b.String()
}

// Refused shows every candidate and what happened to each. It never substitutes.
func Refused(a Agent, tries []Try) string {
	var b strings.Builder
	b.WriteString(head("install " + a.Name))
	b.WriteString("  " + bright.Render("cannot build "+a.Name+" — every pin has moved") + "\n")
	b.WriteString(rule())
	for _, t := range tries {
		b.WriteString("  " + pad(dim.Render(t.Pin.Ref), 44) + pad(dim.Render(string(t.Outcome)), 16) + dim.Render(t.Detail) + "\n")
	}
	for _, t := range tries {
		if t.Outcome == DigestWrong {
			b.WriteString("\n  " + dim.Render("One of those downloads, but it is not the file "+strings.ToUpper(a.Name)+" was") + "\n")
			b.WriteString("  " + dim.Render("tested against. Building on it would give you something that") + "\n")
			b.WriteString("  " + dim.Render("answers to the name and has never been checked.") + "\n")
			break
		}
	}
	b.WriteString("\n  " + bright.Render("So this stops. An agent you cannot verify is not the agent.") + "\n")
	b.WriteString("\n  " + quiet.Render("by manymoats") + "\n")
	return b.String()
}

// Held renders a verify run. Untested is its own block and says out loud that
// it is not a pass — the rule every surface in this house follows.
func Held(a Agent, results []Result) string {
	var b strings.Builder
	b.WriteString(head("verify " + a.Name))
	b.WriteString("  " + dim.Render(pad("claim", 36)+pad("asked", 13)+"held") + "\n")
	b.WriteString(rule())
	broke := 0
	for _, r := range results {
		v := good.Render(fmt.Sprintf("%d of %d", r.Held, r.Asked))
		if r.Held < r.Asked {
			v = bright.Bold(true).Render(fmt.Sprintf("%d of %d", r.Held, r.Asked))
			broke++
		}
		// Clip the plain text, then style it. Clipping a styled string cuts
		// inside the escape sequence; padding one that is already too long
		// silently runs the next column into it.
		asked := fmt.Sprintf("%d ways", r.Asked)
		if r.Runs > 1 {
			asked = fmt.Sprintf("%d ×%d", r.Asked/r.Runs, r.Runs)
		}
		b.WriteString("  " + pad(bright.Render(clip(r.Says, 35)), 36) + pad(dim.Render(asked), 13) + v + "\n")
		// The evidence gets its own line. Appended to the probe it was clipped
		// away by the line budget, which put a verdict on the page beside a
		// quote too short to support it.
		for _, f := range r.Failures {
			probe, why, split := strings.Cut(f, " → ")
			if !split {
				b.WriteString("  " + quiet.Render("    "+clip(f, 66)) + "\n")
				continue
			}
			b.WriteString("  " + quiet.Render("    "+clip(probe, 66)) + "\n")
			b.WriteString("  " + dim.Render("      "+clip(why, 64)) + "\n")
		}
	}
	if len(a.Known) > 0 {
		b.WriteString("\n  " + bright.Render("measured, and true anyway") + "\n")
		b.WriteString(rule())
		for _, k := range a.Known {
			b.WriteString("  " + dim.Render(k) + "\n")
		}
	}
	if len(a.Untested) > 0 {
		b.WriteString("\n  " + bright.Render("what this did not test") + "\n")
		b.WriteString(rule())
		for _, u := range a.Untested {
			b.WriteString("  " + dim.Render(u) + "\n")
		}
		b.WriteString("\n  " + dim.Render(fmt.Sprintf("%d untested · they are not passes", len(a.Untested))) + "\n")
	}
	if broke > 0 {
		b.WriteString("\n  " + bright.Bold(true).Render(fmt.Sprintf("%d of %d claims did not hold on this machine, at this pin.", broke, len(results))) + "\n")
	} else if len(results) > 0 {
		b.WriteString("\n  " + good.Render(fmt.Sprintf("all %d claims held", len(results))) + dim.Render(" — the tested ones. The block above is not covered.") + "\n")
	}
	b.WriteString("\n  " + quiet.Render("by manymoats") + "\n")
	return b.String()
}

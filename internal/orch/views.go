package orch

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/manymoats/manymoats/internal/agent"
)

func hue(a agent.Agent) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(agent.MarkFor(a.Source).Color))
}

func inverted(a agent.Agent, s string) string {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(agent.MarkFor(a.Source).Color)).
		Foreground(lipgloss.Color("#0a0d10")).Bold(true).Render(s)
}

func live(a agent.Agent) bool { return a.State == agent.Working || a.State == agent.Asks }

// pad measures with lipgloss.Width, which strips ANSI escapes AND counts
// double-width runes correctly. runewidth alone counted escape bytes as visible
// columns — a latent bug that only surfaced when a two-cell emoji arrived.
func pad(s string, n int) string {
	w := lipgloss.Width(s)
	if w >= n {
		return s
	}
	return s + strings.Repeat(" ", n-w)
}

func (m model) header(title string) string {
	return "  " + dim.Render(title) + strings.Repeat(" ", 30) + dim.Render(time.Now().Format("15:04:05")) + "\n\n"
}

// footer says only what is true and non-zero. "0 waiting" is noise dressed as
// information — if nothing is waiting, the right word count is zero.
func (m model) footer() string {
	w, a, hidden := 0, 0, 0
	for _, x := range m.agents {
		switch x.State {
		case agent.Working:
			w++
		case agent.Asks:
			a++
		default:
			hidden++
		}
	}
	var parts []string
	if w > 0 {
		parts = append(parts, fmt.Sprintf("%d working", w))
	}
	if a > 0 {
		parts = append(parts, lipgloss.NewStyle().Bold(true).Render(plural(a, "waiting on you", "waiting on you")))
	}
	if len(parts) == 0 {
		parts = append(parts, dim.Render("all clear"))
	}
	if hidden > 0 && !m.showAll {
		parts = append(parts, dim.Render(fmt.Sprintf("%d idle", hidden)))
	}

	var b strings.Builder
	b.WriteString("  " + dim.Render(strings.Repeat("─", 52)) + "\n")
	b.WriteString("  " + strings.Join(parts, dim.Render("   ")) + "\n")

	var seen []string
	sawSrc := map[agent.Source]bool{}
	for _, x := range m.visible() {
		if sawSrc[x.Source] {
			continue
		}
		sawSrc[x.Source] = true
		seen = append(seen, hue(x).Bold(true).Render(agent.MarkFor(x.Source).Tiny())+dim.Render(" "+agent.Brand(x.Source)))
	}
	if len(seen) > 0 {
		b.WriteString("  " + strings.Join(seen, "   ") + "\n")
	}
	b.WriteString("  " + dim.Render("1-4 views · m minimal · h machines · a old · n names · q quit") +
		"  " + maker() + "\n")
	return b.String()
}

// maker is always on screen — quiet, at the end of the keys line, never in the
// way of the data. The founder asked for it to be permanent rather than
// first-run only.
// minimalKeys is the way out. A strip you leave open all day must still say how
// to leave it — being stuck with no visible exit is worse than a little noise.
func minimalKeys() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#5c6673")).Render("m back · q quit")
}

func maker() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#3a4450")).Render("by manymoats")
}

// visible defaults to what is actually happening. Everything else — idle
// sessions, resident models, hours-old stalls — is one keypress away.
func (m model) visible() []agent.Agent {
	if m.showAll {
		return m.agents
	}
	return agent.OnlyActive(m.agents)
}

func colWidths(as []agent.Agent, mode agent.NameMode) (int, int) {
	mw, pw := 10, 10
	for _, a := range as {
		if w := lipgloss.Width(labelFor(a, mode)); w > mw {
			mw = w
		}
		if w := lipgloss.Width(a.Project); w > pw {
			pw = w
		}
	}
	if mw > 24 {
		mw = 24
	}
	if pw > 20 {
		pw = 20
	}
	return mw, pw
}

func clipNo(s string, n int) string { return agent.Shorten(s, n) }

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func (m model) instrument() string {
	var b strings.Builder
	b.WriteString(m.header("ORCH · instrument"))
	vis := m.visible()
	mw, pw := colWidths(vis, m.names)
	for i, a := range vis {
		if i >= 14 {
			break
		}
		c := hue(a)
		if !live(a) {
			c = c.Faint(true)
		}
		_, mid, _ := agent.MarkFor(a.Source).Render()
		row := fmt.Sprintf("  %s %s %s ", pad(c.Render(mid), 2), pad(clipNo(labelFor(a, m.names), mw), mw), pad(agent.ShortProject(a.Project, pw), pw))
		b.WriteString(row)
		if a.State == agent.Asks {
			b.WriteString(inverted(a, " WAITING ON YOU "))
		} else {
			const rateCells = 8
			r := strings.Repeat(" ", rateCells)
			if live(a) && a.TokensMin > 0 {
				r = pad(rate(a.TokensMin)+"/m", rateCells)
			} else if a.CPUPct > 0 {
				r = pad(fmt.Sprintf("%5.1f%% cpu", a.CPUPct), rateCells)
			}
			pot := a.Pot
			if pot == "" {
				pot = "—"
			}
			b.WriteString(c.Render(breathe(meterFor(a), m.frame, live(a))) + dim.Render(r) +
				dim.Render(fmt.Sprintf(" %-9s %5s  %-11s", a.State, short(a.Since), pot)))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n" + m.footer())
	return b.String()
}

func wave(frame, seed int, amp float64) string {
	const glyphs = "⣀⣄⣤⣦⣶⣷⣿"
	r := []rune(glyphs)
	var s strings.Builder
	for x := 0; x < 26; x++ {
		v := (sin(float64(x)*0.42+float64(frame)*0.3+float64(seed)) + 1) / 2
		i := int(v * amp)
		if i > 6 {
			i = 6
		}
		if i < 0 {
			i = 0
		}
		s.WriteRune(r[i])
	}
	return s.String()
}

func sin(x float64) float64 {
	for x > 6.28318 {
		x -= 6.28318
	}
	x2 := x * x
	return x * (1 - x2/6 + x2*x2/120)
}

func (m model) waveform() string {
	var b strings.Builder
	b.WriteString(m.header("ORCH · waveform"))
	for i, a := range m.agents {
		if i >= 6 {
			break
		}
		c := hue(a)
		_, mid, _ := agent.MarkFor(a.Source).Render()
		b.WriteString("  " + c.Bold(true).Render(mid) + " " + c.Bold(live(a)).Render(labelFor(a, m.names)) + dim.Render(" · "+a.Project) + "\n")
		if a.State == agent.Asks {
			b.WriteString("  " + inverted(a, " WAITING ON YOU ") + "\n\n")
			continue
		}
		if live(a) {
			b.WriteString("  " + c.Render(wave(m.frame, i, waveAmp(a.TokensMin))) + "\n\n")
		} else {
			b.WriteString("  " + dim.Render(strings.Repeat("─", 26)+"  "+a.State.String()) + "\n\n")
		}
	}
	b.WriteString(m.footer())
	return b.String()
}

func (m model) cards() string {
	var b strings.Builder
	b.WriteString(m.header("ORCH · cards"))
	const w = 28
	for i, a := range m.agents {
		if i >= 6 {
			break
		}
		c := hue(a)
		tl, tr, bl, br, h, v := "┌", "┐", "└", "┘", "─", "│"
		if a.State == agent.Asks {
			tl, tr, bl, br, h, v = "╔", "╗", "╚", "╝", "═", "║"
		}
		if !live(a) {
			c = c.Faint(true)
		}
		_, mid, _ := agent.MarkFor(a.Source).Render()
		b.WriteString("  " + c.Render(tl+strings.Repeat(h, w)+tr) + "\n")
		b.WriteString("  " + c.Render(v) + c.Bold(true).Render(pad(clipNo(" "+mid+" "+labelFor(a, m.names), w), w)) + c.Render(v) + "\n")
		b.WriteString("  " + c.Render(v) + dim.Render(pad(" "+a.Project, w)) + c.Render(v) + "\n")
		if a.State == agent.Asks {
			b.WriteString("  " + c.Render(v) + inverted(a, pad(" WAITING ON YOU", w)) + c.Render(v) + "\n")
		} else {
			bar, val := Reading(a)
			bar = breathe(bar, m.frame, live(a))
			line := fmt.Sprintf(" %s  %s", c.Render(bar), val)
			b.WriteString("  " + c.Render(v) + pad(line, w) + c.Render(v) + "\n")
		}
		b.WriteString("  " + c.Render(v) + dim.Render(pad(fmt.Sprintf(" %s · %s", a.State, short(a.Since)), w)) + c.Render(v) + "\n")
		b.WriteString("  " + c.Render(bl+strings.Repeat(h, w)+br) + "\n\n")
	}
	b.WriteString(m.footer())
	return b.String()
}

// minimal is minimal INFORMATION, not minimal visibility. It is the strip left
// open all day, so it must be readable at a glance from across the desk — one
// line, but a full one. Anything that needs the founder leads.
func (m model) minimal() string {
	if m.err != nil {
		return "  " + lipgloss.NewStyle().Bold(true).Render("orch · error") + " " + dim.Render(m.err.Error()) + "\n"
	}
	var alerts, working []string
	for _, a := range m.agents {
		c := hue(a)
		tiny := agent.MarkFor(a.Source).Tiny()
		switch a.State {
		case agent.Asks:
			alerts = append(alerts, inverted(a, " "+tiny+" "+agent.Display(a, agent.NameBrand)+" NEEDS YOU "+short(a.Since)+" "))
		case agent.Working:
			_, val := Reading(a)
			label := c.Bold(true).Render(tiny + " " + agent.Shorten(agent.Display(a, m.names), 12))
			if val != "" {
				label += dim.Render(" " + val)
			}
			working = append(working, label)
		}
	}
	if len(alerts) == 0 && len(working) == 0 {
		return "  " + dim.Render("orch") + "  " +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#5FD0C0")).Render("all clear") +
			"   " + minimalKeys() + "   " + maker() + "\n"
	}
	var b strings.Builder
	b.WriteString("  ")
	if len(alerts) > 0 {
		b.WriteString(strings.Join(alerts, " ") + "  ")
	}
	b.WriteString(strings.Join(working, dim.Render("  ·  ")))
	if m.showHost {
		for _, st := range m.hosts {
			if st.Reachable && st.GPUPct > 0 {
				b.WriteString(dim.Render(fmt.Sprintf("   %s gpu %.0f%%", st.Name, st.GPUPct)))
			}
		}
	}
	b.WriteString("   " + minimalKeys() + "   " + maker() + "\n")
	return b.String()
}

func rate(tpm float64) string {
	switch {
	case tpm <= 0:
		return "      "
	case tpm >= 1000:
		return fmt.Sprintf("%5.1fk", tpm/1000)
	default:
		return fmt.Sprintf("%6.0f", tpm)
	}
}

func waveAmp(tokensPerMin float64) float64 {
	if tokensPerMin <= 0 {
		return 0.4
	}
	a := 1 + 5*(tokensPerMin/20000)
	if a > 6 {
		a = 6
	}
	return a
}

func labelFor(a agent.Agent, mode agent.NameMode) string {
	n := agent.Display(a, mode)
	if a.Tag != "" {
		n += "·" + a.Tag
	}
	if a.Subagents > 0 {
		n += subDigit(a.Subagents)
	}
	return n
}

// subDigit marks how many subagents a parent is running, as a superscript so it
// reads as an annotation on the name rather than part of it.
func subDigit(n int) string {
	sup := []string{"", "¹", "²", "³", "⁴", "⁵", "⁶", "⁷", "⁸", "⁹"}
	if n < len(sup) {
		return sup[n]
	}
	return "⁺"
}

func hostBar(p float64, n int) string {
	if p < 0 {
		return strings.Repeat("·", n)
	}
	f := int(p / 100 * float64(n))
	if f > n {
		f = n
	}
	return strings.Repeat("▓", f) + strings.Repeat("░", n-f)
}

func (m model) hostBars() string {
	var b strings.Builder
	silver := lipgloss.NewStyle().Foreground(lipgloss.Color("#A8B3BD"))
	w := 7
	for _, st := range m.hosts {
		if len(st.Name) > w {
			w = len(st.Name)
		}
	}
	for _, st := range m.hosts {
		if !st.Reachable {
			b.WriteString("  " + dim.Render(fmt.Sprintf("%-*s unreachable", w, st.Name)) + "\n")
			continue
		}
		line := fmt.Sprintf("  %-*s cpu %s %3.0f%%   gpu %s %3.0f%%",
			w, st.Name, hostBar(st.CPUPct, 8), st.CPUPct, hostBar(st.GPUPct, 8), st.GPUPct)
		b.WriteString(silver.Render(line) + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

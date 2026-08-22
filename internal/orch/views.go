package orch

import (
	"fmt"
	"math"
	"os"
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

// clockNow is frozen under ORCH_FIXTURE. Two snapshots are two processes, so
// the wall clock advances between them and lands in the diff looking exactly
// like an animated digit — evidence that cannot tell motion from time is
// useless, and this is the second time that trap has cost a review round.
func clockNow() string {
	if os.Getenv("ORCH_FIXTURE") != "" {
		return "00:00:00"
	}
	return time.Now().Format("15:04:05")
}

// pulse is the whole liveness signal now: one cell, carrying no value, in every
// view's header. Anything that encodes a number holds still.
func (m model) pulse() string {
	live := false
	for _, a := range m.agents {
		if a.State == agent.Working || a.State == agent.Asks {
			live = true
			break
		}
	}
	return dim.Render(heartbeat(m.frame, live))
}

func (m model) header(title string) string {
	// The pulse sits against the clock on purpose. Floating beside the title it
	// was an unlabelled dot — the adversary seat was right that it read as decor,
	// because its meaning lived in my prose and not on the surface. Next to a
	// running clock it needs no label: a clock says "now", and a pulse beside it
	// says "still".
	return "  " + dim.Render(title) + strings.Repeat(" ", 29) +
		m.pulse() + " " + dim.Render(clockNow()) + "\n\n"
}

// footer says only what is true and non-zero. "0 waiting" is noise dressed as
// information — if nothing is waiting, the right word count is zero.
func (m model) footer() string {
	w, a := 0, 0
	for _, x := range m.agents {
		switch x.State {
		case agent.Working:
			w++
		case agent.Asks:
			a++
		}
	}
	// Count what is actually off screen. Bucketing done/stalled/idle/resident by
	// state and calling the total "idle" labelled rows that were visible, with a
	// word that was wrong for three of the four.
	hidden := len(m.agents) - len(m.visible())
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
		parts = append(parts, dim.Render(fmt.Sprintf("%d not shown", hidden)))
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
// heartbeat is the whole liveness signal for the minimal strip: one cell,
// carrying no value, next to nothing it could be mistaken for.
// clockShort is minutes only. Seconds cost two cells the strip does not have,
// and a strip you glance at does not need them.
func clockShort() string {
	if os.Getenv("ORCH_FIXTURE") != "" {
		return "00:00"
	}
	return time.Now().Format("15:04")
}

func heartbeat(frame int, alive bool) string {
	if !alive {
		return " "
	}
	return string([]rune("·••·")[(frame/3)%4])
}

// stripStyle measures what a styled string will actually occupy.
func stripStyle(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

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
			const rateCells = 10
			r := strings.Repeat(" ", rateCells)
			if live(a) && a.TokensMin > 0 {
				r = pad(strings.TrimSpace(rate(a.TokensMin))+"/m", rateCells)
			} else if a.CPUPct > 0 {
				r = pad(cores(a.CPUPct), rateCells)
			}
			pot := a.Pot
			if pot == "" {
				pot = "—"
			}
			b.WriteString(c.Render(meterFor(a)) + " " + dim.Render(r) +
				dim.Render(fmt.Sprintf(" %-9s %5s  %-11s", a.State, short(a.Since), pot)))
		}
		b.WriteString("\n")
	}
	b.WriteString(more(len(vis), 14) + "\n" + m.footer())
	return b.String()
}

// more says out loud that the board is not showing everything. A count in the
// footer that disagrees with the rows on screen reads as "this is all of them".
func more(total, shown int) string {
	if total <= shown {
		return ""
	}
	return "  " + dim.Render(fmt.Sprintf("+%d more", total-shown)) + "\n"
}

// trace draws the samples actually recorded, oldest on the left. A position on
// this line is a moment that happened, so the shape only changes when a new
// reading arrives — the past holds still, because the past is fixed.
func trace(h []float64, _ int, _ bool) string {
	const cells = 26
	glyphs := []rune("⣀⣄⣤⣦⣶⣷⣿")
	if len(h) == 0 {
		return strings.Repeat("·", cells)
	}
	var s strings.Builder
	// Not yet a full window: the empty part reads as "no reading yet", not zero.
	for i := 0; i < cells-len(h); i++ {
		s.WriteRune('·')
	}
	for n, v := range h {
		// A window-relative scale would redraw every historical cell the moment a
		// new high arrived — the past changing because the future did. The scale
		// is fixed, and it is the same one the meter uses, so a cell here and a
		// bar there mean the same thing.
		i := traceLevel(v, len(glyphs))
		if i < 0 {
			i = 0
		}
		if i >= len(glyphs) {
			i = len(glyphs) - 1
		}
		_ = n
		s.WriteRune(glyphs[i])
	}
	return s.String()
}

// traceLevel maps a rate to a glyph index on a FIXED log scale — the same
// 50k/min ceiling meterFor uses. Fixed is what makes recorded history stable.
func traceLevel(v float64, levels int) int {
	if v <= 0 {
		return 0
	}
	f := int(math.Round(math.Log10(v+1) / math.Log10(50000) * float64(levels-1)))
	if f >= levels {
		f = levels - 1
	}
	return f
}

func wave(frame, seed int, amp float64) string {
	const glyphs = "⣀⣄⣤⣦⣶⣷⣿"
	r := []rune(glyphs)
	var s strings.Builder
	for x := 0; x < 26; x++ {
		v := (math.Sin(float64(x)*0.42+float64(frame)*0.3+float64(seed)) + 1) / 2
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

func (m model) waveform() string {
	var b strings.Builder
	b.WriteString(m.header("ORCH · waveform"))
	vis := m.visible()
	for i, a := range vis {
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
		if live(a) && a.TokensMin > 0 {
			_, val := Reading(a)
			b.WriteString("  " + c.Render(trace(m.history[a.ID], m.frame, live(a))) + "  " + dim.Render(val) + "\n\n")
		} else if live(a) {
			// Cursor and the local models expose no token rate. A wave here would
			// be a picture of a number nobody measured.
			_, val := Reading(a)
			b.WriteString("  " + dim.Render(strings.Repeat("·", 26)) + "  " + dim.Render(val) + "\n\n")
		} else {
			b.WriteString("  " + dim.Render(strings.Repeat("─", 26)+"  "+a.State.String()) + "\n\n")
		}
	}
	b.WriteString(more(len(vis), 6))
	b.WriteString(m.footer())
	return b.String()
}

func (m model) cards() string {
	var b strings.Builder
	b.WriteString(m.header("ORCH · cards"))
	const w = 28
	vis := m.visible()
	// A card for an idle agent costs six rows to say nothing. It is not falsely
	// calm — it is falsely PRESENT, carrying the visual weight of a working agent
	// while conveying no reading. Idle collapses to one dim line so the cards
	// that have something to show get the room.
	var quiet []agent.Agent
	shown := 0
	for _, a := range vis {
		if !live(a) {
			quiet = append(quiet, a)
		}
	}
	for i, a := range vis {
		if !live(a) {
			continue
		}
		if shown >= 6 {
			break
		}
		shown++
		_ = i
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
		b.WriteString("  " + c.Render(v) + dim.Render(pad(clipNo(" "+a.Project, w), w)) + c.Render(v) + "\n")
		if a.State == agent.Asks {
			b.WriteString("  " + c.Render(v) + inverted(a, pad(" WAITING ON YOU", w)) + c.Render(v) + "\n")
		} else {
			bar, val := Reading(a)
			line := fmt.Sprintf(" %s  %s", c.Render(bar), val)
			b.WriteString("  " + c.Render(v) + pad(line, w) + c.Render(v) + "\n")
		}
		b.WriteString("  " + c.Render(v) + dim.Render(pad(fmt.Sprintf(" %s · %s", a.State, short(a.Since)), w)) + c.Render(v) + "\n")
		b.WriteString("  " + c.Render(bl+strings.Repeat(h, w)+br) + "\n\n")
	}
	for _, a := range quiet {
		b.WriteString("  " + dim.Render(fmt.Sprintf("%s · %s · %s",
			agent.Shorten(labelFor(a, m.names), 14), agent.ShortProject(a.Project, 12), a.State)) + "\n")
	}
	if len(quiet) > 0 {
		b.WriteString("\n")
	}
	b.WriteString(more(countLive(vis), 6))
	b.WriteString(m.footer())
	return b.String()
}

func countLive(as []agent.Agent) int {
	n := 0
	for _, a := range as {
		if live(a) {
			n++
		}
	}
	return n
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
	// The strip has fixed chrome — the pulse, the clock, the way back, and the
	// maker line the founder asked to always be visible. Clipping from the right
	// ate the maker. So the chrome is reserved and the AGENTS take what is left,
	// dropping to a count when they do not fit. A strip that trims its own name
	// to show one more agent has its priorities backwards.
	width := m.w
	if width <= 0 {
		width = 80
	}
	// Estimating the chrome cost guessed wrong by sixteen cells. Assemble the
	// real line, measure it, drop one agent, repeat — exact, and cheap at this
	// size.
	over := 0
	for {
		if lipgloss.Width(m.stripLine(alerts, working, over)) <= width || len(working) == 0 {
			break
		}
		over++
		working = working[:len(working)-1]
	}
	return m.stripLine(alerts, working, over)
}

// stripLine assembles the one-line strip. The pulse and clock lead, so on a
// narrow terminal the anchor telling you the board is alive is the last thing
// to go, not the first.
func (m model) stripLine(alerts, working []string, over int) string {
	if len(alerts) == 0 && len(working) == 0 {
		return "  " + dim.Render(heartbeat(m.frame, false)) + " " + dim.Render(clockShort()) + "  " +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#5FD0C0")).Render("all clear") +
			"   " + minimalKeys() + "   " + maker() + "\n"
	}
	var b strings.Builder
	// The pulse and clock lead the line. This is the view that gets clipped on a
	// narrow terminal, so the anchor saying the board is alive must be the last
	// thing to go, not the first.
	b.WriteString("  " + dim.Render(heartbeat(m.frame, len(working) > 0 || len(alerts) > 0)) +
		" " + dim.Render(clockShort()) + "  ")
	if len(alerts) > 0 {
		b.WriteString(strings.Join(alerts, " ") + "  ")
	}
	b.WriteString(strings.Join(working, dim.Render(" · ")))
	if over > 0 {
		b.WriteString(dim.Render(fmt.Sprintf("  +%d", over)))
	}
	if m.showHost {
		for _, st := range m.hosts {
			if st.Reachable && st.GPUPct > 0 {
				b.WriteString(dim.Render(fmt.Sprintf("   %s gpu %.0f%%", st.Name, st.GPUPct)))
			}
		}
	}
	// Two spaces, not three. The strip sat one cell from its limit, so a rate
	// one character longer cost a whole agent — a sixteen-cell jump to save one.
	b.WriteString("  " + minimalKeys() + "  " + maker() + "\n")
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
		line := fmt.Sprintf("  %-*s all cores %s %3.0f%%   gpu %s %3.0f%%",
			w, st.Name, hostBar(st.CPUPct, 8), st.CPUPct, hostBar(st.GPUPct, 8), st.GPUPct)
		b.WriteString(silver.Render(line) + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

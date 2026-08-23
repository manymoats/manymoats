package orch

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/manymoats/manymoats/internal/agent"
	"github.com/manymoats/manymoats/internal/collect"
	"github.com/manymoats/manymoats/internal/conf"
	"github.com/manymoats/manymoats/internal/host"
	"github.com/muesli/termenv"
)

type tickMsg time.Time

type view int

const (
	viewAnim   view = iota
	viewSplash      // last still of the same cut — not a second movie
	viewMarks
	viewInstrument
	viewWaveform
	viewCards
	viewMinimal
)

func (v view) String() string {
	return [...]string{"anim", "splash", "marks", "instrument", "waveform", "cards", "minimal"}[v]
}

type model struct {
	view        view
	names       agent.NameMode
	animLong    bool
	animElapsed int
	gotData     bool
	showAll     bool
	showHost    bool
	hosts       []host.Stats
	agents      []agent.Agent
	history     map[string][]float64
	frame       int
	w, h        int
	err         error
}

// histCap is how many samples the trace keeps — one per data tick, so 26
// seconds of history at the one-second data clock.
const histCap = 26

// record appends this tick's reading. The trace draws THIS, not a synthetic
// wave: a picture that scrolls like a history while being generated from the
// current value implies the past is changing, which is a lie about data.
func (m *model) record(as []agent.Agent) {
	if m.history == nil {
		m.history = map[string][]float64{}
	}
	seen := map[string]bool{}
	for _, a := range as {
		seen[a.ID] = true
		v := a.TokensMin
		h := append(m.history[a.ID], v)
		if len(h) > histCap {
			h = h[len(h)-histCap:]
		}
		m.history[a.ID] = h
	}
	for id := range m.history {
		if !seen[id] {
			delete(m.history, id)
		}
	}
}

// seedFixtureHistory fills the trace window for ORCH_FIXTURE only. It is not a
// measurement and never runs on a real board.
func (m *model) seedFixtureHistory() {
	m.history = map[string][]float64{}
	for _, a := range m.agents {
		if a.TokensMin <= 0 {
			continue
		}
		h := make([]float64, 0, histCap)
		for i := 0; i < histCap; i++ {
			h = append(h, a.TokensMin*(0.55+0.45*float64((i*7)%9)/8))
		}
		m.history[a.ID] = h
	}
}

// Two clocks on purpose. Data refreshes once a second because reading session
// files and running ps more often is waste. Motion runs at 120ms because that is
// what reads as alive. Collapsing them made the board static.
func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

type pulseMsg time.Time

const pulseMS = 120

func pulse() tea.Cmd {
	return tea.Tick(pulseMS*time.Millisecond, func(t time.Time) tea.Msg { return pulseMsg(t) })
}

// probeAmbiguousWide is a WIDTH measurement for East-Asian ambiguous runes.
// It must never be treated as a board-wide "use ASCII" switch — that emptied
// cells whose fallback was a space. Each glyph is decided by GlyphFits.
func probeAmbiguousWide() bool {
	for _, k := range []string{"ORCH_AMBIGUOUS_WIDE", "RUNEWIDTH_EASTASIAN"} {
		switch os.Getenv(k) {
		case "1", "true":
			return true
		case "0", "false":
			return false
		}
	}
	lang := os.Getenv("LC_ALL") + os.Getenv("LC_CTYPE") + os.Getenv("LANG")
	for _, cjk := range []string{"zh", "ja", "ko", "CN", "JP", "KR", "TW"} {
		if strings.Contains(lang, cjk) {
			return true
		}
	}
	return false
}

func (m model) Init() tea.Cmd {
	agent.SetAmbiguousWide(probeAmbiguousWide())
	cmds := []tea.Cmd{tick(), pulse(), refresh}
	if m.view == viewAnim {
		cmds = append(cmds, animTick())
	}
	return tea.Batch(cmds...)
}

type hostMsg struct{ stats []host.Stats }

func hostRefresh() tea.Msg {
	c := conf.Load()
	out := make([]host.Stats, 0, len(c.Machines))
	for _, m := range c.Machines {
		if m.Host == "local" || m.Host == "" {
			out = append(out, host.Local(m.Name))
			continue
		}
		out = append(out, host.Remote(m.Host, m.Name))
	}
	return hostMsg{stats: out}
}

func refresh() tea.Msg {
	as, err := collect.ClaudeAll(collect.ClaudeLive())
	if err != nil {
		return err
	}
	as = append(as, collect.Cursor()...)
	as = append(as, collect.Processes()...)
	as = append(as, collect.Ollama()...)
	as = agent.RollUpSubagents(as)
	as = agent.Settle(as, time.Now())
	for i := range as {
		as[i].Pot, as[i].Free = agent.Paying(as[i].Source, as[i].Model, "")
	}
	agent.Disambiguate(as)
	return as
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyMsg:
		if m.view == viewAnim || m.view == viewSplash {
			s := v.String()
			if s == "ctrl+c" {
				return m, tea.Quit
			}
			if m.view == viewAnim && m.animLong {
				markSeen()
			}
			m.view = viewMarks
			return m, nil
		}
		switch s := v.String(); s {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "1":
			m.view = viewMarks
		case "2":
			m.view = viewInstrument
		case "3":
			m.view = viewWaveform
		case "4":
			m.view = viewCards
		case "m":
			if m.view == viewMinimal {
				m.view = viewMarks
			} else {
				m.view = viewMinimal
			}
		case "tab":
			m.view = viewMarks + (m.view+1)%5
		case "enter":
			if m.view == viewSplash {
				m.view = viewMarks
			}
		case "n":
			m.names = (m.names + 1) % 3
		case "a":
			m.showAll = !m.showAll
		case "h":
			m.showHost = !m.showHost
			if m.showHost {
				return m, hostRefresh
			}
		}
	case tea.WindowSizeMsg:
		m.w, m.h = v.Width, v.Height
		agent.SetAmbiguousWide(probeAmbiguousWide())
	case []agent.Agent:
		m.agents = v
		m.gotData = true
		m.record(v)
		// The picture yields to the data. If the short cut is still running when
		// real state arrives, finish the sweep where it is and show the board.
		if m.view == viewAnim && !m.animLong && m.animElapsed >= 450 {
			m.view = viewMarks
		}
	case hostMsg:
		m.hosts = v.stats
	case error:
		m.err = v
	case animFrame:
		if m.view != viewAnim {
			return m, nil
		}
		m.animElapsed += frameMS
		m.frame++
		if m.animDone() {
			if m.animLong {
				markSeen()
			}
			m.view = viewMarks
			return m, nil
		}
		return m, animTick()
	case pulseMsg:
		m.frame++
		return m, pulse()
	case tickMsg:
		return m, tea.Batch(tick(), refresh)
	}
	return m, nil
}

var dim = lipgloss.NewStyle().Foreground(lipgloss.Color("#4c5865"))

const meterCells = 7

// Reading returns the bar AND the number that produced it. Two agents can be
// measured in different units — Claude reports tokens/min, Cursor exposes only a
// CPU share, a resident model reports neither — so a bare bar invites a
// comparison that does not exist. The number is the truth; the bar is a glance.
func Reading(a agent.Agent) (bar, value string) {
	switch {
	case a.TokensMin > 0:
		return meterFor(a), fmt.Sprintf("%s/m", trimRate(a.TokensMin))
	case a.CPUPct > 0:
		return meterFor(a), cores(a.CPUPct)
	case a.LinesTouched > 0:
		return meterFromLines(a), fmt.Sprintf("%s lines", compact(a.LinesTouched))
	case a.VRAMBytes > 0:
		return dim.Render(strings.Repeat("·", meterCells)), fmt.Sprintf("%.1fGB", float64(a.VRAMBytes)/1e9)
	default:
		return dim.Render(strings.Repeat("·", meterCells)), ""
	}
}

// cores says what the number actually measures. `ps %cpu` is a share of ONE
// core, so a busy agent routinely reads 300%; the machine rows below it report
// a share of the WHOLE machine. Both said "cpu", ten times apart, and neither
// said which — so the board had two different units under one word.
func cores(pct float64) string {
	if pct >= 100 {
		return fmt.Sprintf("%.1f cores", pct/100)
	}
	return fmt.Sprintf("%.0f%% core", pct)
}

func compact(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func meterFromLines(a agent.Agent) string {
	if !live(a) {
		return dim.Render(strings.Repeat("·", meterCells))
	}
	f := int(math.Round(math.Log10(float64(a.LinesTouched)+1) / math.Log10(50000) * meterCells))
	if f < 1 {
		f = 1
	}
	if f > meterCells {
		f = meterCells
	}
	return strings.Repeat("▓", f) + strings.Repeat("░", meterCells-f)
}

func trimRate(t float64) string {
	if t >= 1000 {
		return fmt.Sprintf("%.1fk", t/1000)
	}
	return fmt.Sprintf("%.0f", t)
}

func meterFor(a agent.Agent) string {
	if a.TokensMin > 0 {
		return meter(a.TokensMin, live(a))
	}
	if a.CPUPct > 0 {
		// full scale is a whole core. Scaling to a third pinned the bar at 100%
		// for anything above 33%, so a 74%% reading drew as full.
		f := int(math.Round(a.CPUPct / 100 * meterCells))
		if f < 1 {
			f = 1
		}
		if f >= meterCells {
			// Over full scale. A plain full bar would claim "exactly one core".
			return strings.Repeat("▓", meterCells-1) + "█"
		}
		return strings.Repeat("▓", f) + strings.Repeat("░", meterCells-f)
	}
	return dim.Render(strings.Repeat("·", meterCells))
}

// breathe is gone. It shaded the frontier cell of the meter, and the adversary
// seat was right that this alters a payload: a reader counting filled cells sees
// ▓ where the level is ░, which is one whole division. The number never moved,
// but the bar is data too. Liveness now comes from heartbeat() — one cell that
// encodes nothing, so it can be animated freely.

func meter(tokensPerMin float64, active bool) string {
	if !active || tokensPerMin <= 0 {
		return dim.Render(strings.Repeat("░", meterCells))
	}
	filled := int(math.Round(math.Log10(tokensPerMin+1) / math.Log10(50000) * meterCells))
	if filled < 1 {
		filled = 1
	}
	if filled >= meterCells {
		return strings.Repeat("▓", meterCells-1) + "█"
	}
	return strings.Repeat("▓", filled) + strings.Repeat("░", meterCells-filled)
}

// fit is the last guard before anything reaches the screen. Only the marks
// board reflows to the terminal width; every other view is laid out at a fixed
// size, and on a narrow terminal an over-long line wraps into garbage that
// looks like a rendering bug. Clipping is not a layout — it is the promise that
// the board never spills.
func fit(s string, w int) string {
	if w <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if lipgloss.Width(l) > w {
			lines[i] = clipStyled(l, w)
		}
	}
	return strings.Join(lines, "\n")
}

// clipStyled truncates to w DISPLAY CELLS without ever cutting inside an escape
// sequence. The first version measured with lipgloss.Width and then cut with a
// rune-counting shortener, which counted escape bytes as content and sliced
// through them — the line came out as six cells of garbage. Same bug the house
// already paid for once with pad().
func clipStyled(s string, w int) string {
	var b strings.Builder
	seen := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			j := i
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++
			}
			b.WriteString(s[i:j]) // styles cost no cells and are never cut
			i = j
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		rw := lipgloss.Width(string(r))
		if seen+rw > w {
			break
		}
		b.WriteRune(r)
		seen += rw
		i += size
	}
	return b.String() + "\x1b[0m"
}

func (m model) View() string {
	if m.err != nil {
		return "\n  " + lipgloss.NewStyle().Bold(true).Render("orch · error") + "\n  " +
			dim.Render(m.err.Error()) + "\n\n  " + dim.Render("q quit") + "\n"
	}
	var out string
	switch m.view {
	case viewAnim:
		out = m.animView()
	case viewSplash:
		out = m.splash()
	case viewInstrument:
		out = m.instrument()
	case viewWaveform:
		out = m.waveform()
	case viewCards:
		out = m.cards()
	case viewMinimal:
		out = m.minimal()
	default:
		out = m.marks()
	}
	return fit(out, m.w)
}

// marks lays agents ACROSS the screen, not down it. When everything is in one
// project nothing is labelled — repeating the same folder on every row is noise.
// With several projects the agents group under a coloured folder, so a glance at
// the colour tells you where without reading a word.
func (m model) marks() string {
	var b strings.Builder
	b.WriteString(m.header(""))
	vis := m.visible()
	single := agent.SingleProject(vis)
	order, byProject := agent.GroupByProject(vis)

	b.WriteString(m.markRows(vis, !single))
	// the key: colour → project, printed once, only when there is more than one
	if !single {
		var legend []string
		for _, proj := range order {
			ph := lipgloss.NewStyle().Foreground(lipgloss.Color(agent.ProjectHue(proj)))
			legend = append(legend, ph.Render(agent.FolderMark)+" "+dim.Render(agent.ShortProject(proj, 18)))
		}
		b.WriteString("  " + strings.Join(legend, "   ") + "\n\n")
	} else if len(order) == 1 && order[0] != agent.UnknownProject {
		ph := lipgloss.NewStyle().Foreground(lipgloss.Color(agent.ProjectHue(order[0])))
		b.WriteString("  " + ph.Render(agent.FolderMark) + " " + dim.Render(agent.ShortProject(order[0], 22)) + "\n\n")
	}
	_ = byProject
	if m.showHost {
		b.WriteString(m.hostBars())
	}
	b.WriteString(m.footer())
	return b.String()
}

func (m model) markRows(as []agent.Agent, showFolders bool) string {
	var b strings.Builder
	// Disambiguate() mints a 4-char tag precisely so two same-model agents in one
	// project can be told apart. A fixed 13 cut it to 3, so the board rendered a
	// wrong identifier and the two columns looked identical.
	cw := 13
	for _, a := range as {
		if w := lipgloss.Width(labelFor(a, m.names)) + 3; w > cw {
			cw = w
		}
	}
	if cw > 20 {
		cw = 20
	}
	lead := "  "
	perRow := 5
	if m.w > 0 {
		perRow = (m.w - len(lead) - 2) / cw
	}
	if perRow < 1 {
		perRow = 1
	}
	for start := 0; start < len(as); start += perRow {
		end := start + perRow
		if end > len(as) {
			end = len(as)
		}
		var l1, lf, l2, l3, l4 strings.Builder
		for _, a := range as[start:end] {
			c := hue(a)
			if !live(a) {
				c = c.Faint(true)
			}
			_, mid, _ := agent.MarkFor(a.Source).Render()
			l1.WriteString("  " + pad(c.Bold(true).Render(mid), cw-2))
			if showFolders {
				ph := lipgloss.NewStyle().Foreground(lipgloss.Color(agent.ProjectHue(a.Project)))
				lf.WriteString("  " + pad(ph.Render(agent.FolderMark), cw-2))
			}
			l2.WriteString("  " + pad(c.Bold(live(a)).Render(agent.Shorten(labelFor(a, m.names), cw-3)), cw-2))
			if a.State == agent.Asks {
				l3.WriteString("  " + pad(inverted(a, "needs you"), cw-2))
				l4.WriteString("  " + pad(dim.Render(short(a.Since)), cw-2))
			} else if !live(a) {
				// Without this, a stalled agent and an idle one both rendered as
				// dots with nothing under them — identical in monochrome, and the
				// difference between "died six hours ago" and "never started" is
				// exactly the one worth noticing.
				l3.WriteString("  " + pad(dim.Render(a.State.String()), cw-2))
				// "0s" under an idle agent is a zero dressed as information —
				// the founder's own rule: if there is nothing, show nothing.
				age := ""
				if a.Since > 0 {
					age = short(a.Since)
				}
				l4.WriteString("  " + pad(dim.Render(age), cw-2))
			} else {
				bar, val := Reading(a)
				l3.WriteString("  " + pad(c.Render(bar), cw-2))
				l4.WriteString("  " + pad(dim.Render(val), cw-2))
			}
		}
		rows := []*strings.Builder{&l1}
		if showFolders {
			rows = append(rows, &lf)
		}
		rows = append(rows, &l2, &l3, &l4)
		for _, l := range rows {
			b.WriteString(lead + strings.TrimRight(l.String(), " ") + "\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func short(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%.0fs", d.Seconds())
	case d < time.Hour:
		return fmt.Sprintf("%.0fm", d.Minutes())
	default:
		return fmt.Sprintf("%.0fh", d.Hours())
	}
}

func namesFromEnv() agent.NameMode {
	switch os.Getenv("ORCH_NAMES") {
	case "brand", "1":
		return agent.NameBrand
	case "both", "2":
		return agent.NameBoth
	default:
		return agent.NameModel
	}
}

// animFrames prints the long cut as still frames so it can be reviewed without a
// terminal — the same reason --snapshot exists.
func animFrames(long bool, n int) int {
	if n < 1 {
		n = 8
	}
	for i := 0; i <= n; i++ {
		var f int
		if long {
			f = 31 * i / n
		} else {
			f = 24 + 7*i/n
		}
		fmt.Printf("\n── frame %02d ─────────────────────────────────────\n%s", f+1, RenderCut(f, 80, 40))
	}
	return 0
}

// printIcons exists because the desk cannot see whether a Nerd Font renders on
// your machine — only you can. Tool icons (play, folder, pulse) may be Nerd
// Font. Provider marks stay unicode so they never tofu.
func setIcons(on bool) int {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "orch:", err)
		return 1
	}
	p := filepath.Join(home, ".orch", "config.json")
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	c := map[string]any{}
	if b, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(b, &c)
	}
	c["icons"] = map[bool]string{true: "nerd", false: "unicode"}[on]
	b, _ := json.MarshalIndent(c, "", "  ")
	if err := os.WriteFile(p, append(b, '\n'), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "orch:", err)
		return 1
	}
	fmt.Printf("  icons: %s\n", c["icons"])
	return 0
}

func printIcons() {
	fmt.Printf("\n  Nerd Font file installed: %v\n", agent.NerdFontInstalled())
	fmt.Printf("  Icons currently: %s\n", map[bool]string{true: "nerd", false: "plain"}[agent.UseNerd()])
	fmt.Println("\n  tool     nerd   ascii")
	fmt.Println("  ─────────────────────")
	for _, ic := range agent.AllToolIcons() {
		shown := ic.ASCII
		if agent.UseNerd() && agent.NerdFontInstalled() && agent.GlyphFits(ic.Nerd) {
			shown = ic.Nerd
		}
		fmt.Printf("  %-8s %s      %s\n", ic.Name, shown, ic.ASCII)
	}
	fmt.Println("\n  Providers stay on unicode shapes (✶ ◧ ◆). Those always draw.")
	fmt.Println("  A boxed nerd glyph is a defect — we print the ascii instead.")
	fmt.Println()
	if agent.NerdFontInstalled() && !agent.UseNerd() {
		fmt.Println("  Font is on disk. Turn icons on:  manymoats orch --icons-on")
		fmt.Println("  The terminal font must name " + nerdFamily + ".")
	}
	if !agent.NerdFontInstalled() {
		fmt.Println("  No Nerd Font on disk. orch keeps using ascii/unicode.")
		fmt.Println("  manymoats orch setup can install " + nerdFamily + ".")
	}
	fmt.Println()
}

// frameFromEnv lets a snapshot render any point in the motion cycle. Without it
// every capture is frame 0, so anyone reviewing a still has to GUESS what moves —
// and a reviewer who guessed once told us the numbers animate, which they do not.
func frameFromEnv() int {
	n, err := strconv.Atoi(os.Getenv("ORCH_FRAME"))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// fixtureAgents is a frozen board. Motion evidence captured from live data is
// worthless: each snapshot is a new process that re-reads cpu and token rates,
// so the numbers move between captures for reasons that have nothing to do with
// animation. With a fixture, anything that differs between two frames is the
// animation and nothing else.
func fixtureAgents() []agent.Agent {
	return []agent.Agent{
		{Source: agent.Claude, Model: "opus-5", Project: "orch", State: agent.Working, TokensMin: 10400},
		{Source: agent.Cursor, Model: "cursor", Project: "manymoats", State: agent.Working, CPUPct: 236.1},
		{Source: agent.Ollama, Model: "big", Project: "manymoats", State: agent.Idle},
	}
}

func snapshot(v view) int {
	m := model{w: 80, h: 40, view: v, names: namesFromEnv(), showHost: os.Getenv("ORCH_HOST") != "", frame: frameFromEnv()}
	if v == viewAnim {
		m.animLong = true
		if os.Getenv("ORCH_FRAME") == "" {
			m.animElapsed = longMS
		} else {
			m.animElapsed = frameFromEnv() * frameMS
		}
	}
	agent.SetAmbiguousWide(probeAmbiguousWide())
	if os.Getenv("ORCH_FIXTURE") != "" {
		m.agents = fixtureAgents()
		m.gotData = true
		// A snapshot has no earlier ticks to draw from, so seed a plausible
		// window. Marked here so nobody mistakes it for a measurement: it exists
		// to make the frozen board reproducible, and only ORCH_FIXTURE reaches it.
		m.seedFixtureHistory()
		fmt.Print(m.View())
		return 0
	}
	switch v := refresh().(type) {
	case []agent.Agent:
		m.agents = v
		m.gotData = true
		m.record(v)
		// The picture yields to the data. If the short cut is still running when
		// real state arrives, finish the sweep where it is and show the board.
		if m.view == viewAnim && !m.animLong && m.animElapsed >= 450 {
			m.view = viewMarks
		}
	case error:
		fmt.Fprintln(os.Stderr, "orch:", v)
		return 1
	}
	if m.showHost {
		if h, ok := hostRefresh().(hostMsg); ok {
			m.hosts = h.stats
		}
	}
	fmt.Print(m.View())
	return 0
}

// Main runs the orch board. It returns an exit code rather than calling
// os.Exit so the manymoats dispatcher owns process teardown.
func Main() int {
	// lipgloss guesses the colour profile from the environment and gets it wrong
	// on some terminals — a board with no colour is a board with no identity, so
	// force TrueColor unless the user has explicitly asked for none.
	if os.Getenv("NO_COLOR") == "" && os.Getenv("ORCH_NO_COLOR") == "" {
		lipgloss.SetColorProfile(termenv.TrueColor)
	}

	args := os.Args[1:]
	for i, a := range args {
		if a == "setup" || a == "--setup" {
			return setup()
		}
		if a == "--icons-on" || a == "--icons-off" {
			return setIcons(a == "--icons-on")
		}
		if a == "--icons" {
			printIcons()
			return 0
		}
		if a == "--anim-frames" {
			long := true
			n := 8
			if i+1 < len(args) && args[i+1] == "short" {
				long = false
			}
			return animFrames(long, n)
		}
		if a == "--snapshot" || a == "-s" {
			v := viewMarks
			if i+1 < len(args) {
				switch args[i+1] {
				case "instrument":
					v = viewInstrument
				case "waveform":
					v = viewWaveform
				case "cards":
					v = viewCards
				case "minimal":
					v = viewMinimal
				case "splash":
					v = viewSplash
				case "anim":
					v = viewAnim
				}
			}
			return snapshot(v)
		}
	}
	start := model{names: namesFromEnv()}
	if motionOK() {
		start.view = viewAnim
		start.animLong = firstRun()
	} else {
		// A run nobody watched must NOT consume the first-run animation. Marking it
		// here burned the once-ever cut on every piped/snapshot invocation.
		// A pipe or CI prints the last still and leaves — tea would wait on a key.
		if !stdoutTTY() || os.Getenv("CI") != "" {
			printLastStill()
			return 0
		}
		// Skip the intro only. The board still runs so the pulse can tick.
		// A still of the cut was already printed for a pipe/CI above.
		start.view = viewMarks
	}
	p := tea.NewProgram(start)
	if _, err := p.Run(); err != nil {
		fmt.Println("orch:", err)
		return 1
	}
	return 0
}

package orch

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/manymoats/manymoats/internal/agent"
)

func render(m model) (s string, panicked any) {
	defer func() { panicked = recover() }()
	s = m.View()
	return
}

func allViews(t *testing.T, m model, label string) {
	t.Helper()
	for v := viewMarks; v <= viewMinimal; v++ {
		m.view = v
		out, p := render(m)
		if p != nil {
			t.Errorf("%s / %s: PANIC %v", label, v, p)
			continue
		}
		if out == "" && v != viewMinimal {
			t.Errorf("%s / %s: rendered nothing", label, v)
		}
	}
}

func TestNoAgentsNeverPanics(t *testing.T) {
	allViews(t, model{w: 80, h: 40}, "no agents")
}

func TestTwoHundredSessions(t *testing.T) {
	var as []agent.Agent
	for i := 0; i < 200; i++ {
		as = append(as, agent.Agent{
			ID: "s", Source: agent.Claude, Model: "claude-opus-5",
			Project: "p", State: agent.Working, Since: time.Second,
		})
	}
	allViews(t, model{w: 80, h: 40, agents: as}, "200 sessions")
}

func TestTinyTerminal(t *testing.T) {
	as := []agent.Agent{{Source: agent.Qwen, Model: "qwen3.8-max", Project: "x", State: agent.Asks, Since: time.Minute}}
	allViews(t, model{w: 20, h: 5, agents: as}, "20 columns")
}

func TestGarbageFieldsNeverPanic(t *testing.T) {
	as := []agent.Agent{
		{Source: "not-a-real-source", Model: "", Project: "", State: agent.Working},
		{Source: agent.Claude, Model: strings.Repeat("x", 500), Project: strings.Repeat("y", 500), State: agent.Stalled, Since: -time.Hour},
	}
	allViews(t, model{w: 80, h: 40, agents: as}, "garbage fields")
}

func TestErrorStateRenders(t *testing.T) {
	m := model{w: 80, h: 40, err: errFake{}}
	out, p := render(m)
	if p != nil {
		t.Fatalf("error state panicked: %v", p)
	}
	if !strings.Contains(out, "error") {
		t.Fatal("an error must be shown, not swallowed")
	}
}

type errFake struct{}

func (errFake) Error() string { return "collector unavailable" }

func TestAsciiFallbackAcrossEveryView(t *testing.T) {
	agent.SetAmbiguousWide(true)
	defer agent.SetAmbiguousWide(false)
	as := []agent.Agent{{Source: agent.Qwen, Model: "qwen", Project: "p", State: agent.Working, Since: time.Second}}
	allViews(t, model{w: 80, h: 40, agents: as}, "wide-ambiguous terminal")
}

func stripANSI(s string) string {
	var out []rune
	skip := false
	for _, r := range s {
		if r == 0x1b {
			skip = true
			continue
		}
		if skip {
			if r == 'm' {
				skip = false
			}
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

func TestInstrumentColumnsNeverDrift(t *testing.T) {
	as := []agent.Agent{
		{Source: agent.Claude, Model: "claude-opus-5", Tag: "191b", Project: "manymoats-kpf", State: agent.Stalled, Since: time.Hour},
		{Source: agent.Claude, Model: "x", Project: "y", State: agent.Working, Since: time.Second, TokensMin: 900},
		{Source: agent.Qwen, Model: strings.Repeat("q", 90), Project: strings.Repeat("p", 90), State: agent.Working, Since: time.Second, TokensMin: 5},
	}
	m := model{w: 100, h: 40, agents: as, view: viewInstrument, showAll: true}
	var widths []int
	for _, ln := range strings.Split(m.View(), "\n") {
		p := stripANSI(ln)
		if strings.Contains(p, "░") || strings.Contains(p, "·······") {
			widths = append(widths, len([]rune(p)))
		}
	}
	if len(widths) < 3 {
		t.Fatalf("expected 3 measurable rows, got %d", len(widths))
	}
	for i := 1; i < len(widths); i++ {
		if widths[i] != widths[0] {
			t.Fatalf("columns drift: %v — a long model name or a session tag must never shift the grid", widths)
		}
	}
}

func TestNoRateShownForSomethingNotRunning(t *testing.T) {
	as := []agent.Agent{
		{Source: agent.Claude, Model: "opus", Project: "p", State: agent.Stalled, Since: 5 * time.Hour, TokensMin: 1400},
		{Source: agent.Claude, Model: "opus", Project: "q", State: agent.Working, Since: time.Second, TokensMin: 1400},
	}
	out := stripANSI(model{w: 100, h: 40, agents: as, view: viewInstrument}.View())
	for _, ln := range strings.Split(out, "\n") {
		if strings.Contains(ln, "stalled") && strings.Contains(ln, "/m") {
			t.Fatalf("a stalled agent must not advertise a live rate: %q", strings.TrimSpace(ln))
		}
	}
	if !strings.Contains(out, "1.4k/m") {
		t.Fatal("a working agent must show its measured rate")
	}
}

func TestNoViewEverHidesACollectorError(t *testing.T) {
	m := model{w: 80, h: 40, err: errFake{}}
	for v := viewSplash; v <= viewMinimal; v++ {
		m.view = v
		out := stripANSI(m.View())
		if strings.Contains(out, "all clear") && !strings.Contains(out, "error") {
			t.Errorf("%s reports 'all clear' while the collector is broken — that is the worst lie a monitor can tell", v)
		}
	}
}

func TestANewViewCannotReintroduceErrorSwallowing(t *testing.T) {
	// The error check lives above the view dispatch. This asserts that property
	// directly, so adding a sixth view cannot quietly bring the bug back.
	m := model{w: 80, h: 40, err: errFake{}}
	for v := view(0); v <= viewMinimal+3; v++ {
		m.view = v
		out := stripANSI(m.View())
		if !strings.Contains(out, "error") {
			t.Fatalf("view %d hides a collector error", v)
		}
	}
}

func TestOnlyAWatchedRunConsumesTheFirstRunCut(t *testing.T) {
	src, _ := os.ReadFile("main.go")
	body := string(src)
	i := strings.Index(body, "if motionOK() {")
	if i < 0 {
		t.Skip("start block moved")
	}
	tail := body[i:]
	if j := strings.Index(tail, "} else {"); j >= 0 {
		elseBlock := tail[j : j+220]
		if strings.Contains(elseBlock, "markSeen()") {
			t.Fatal("the no-motion path marks the first run as seen — a run nobody watched must never burn the once-ever animation")
		}
	}
}

func TestInstallNeverCopiesTheBinary(t *testing.T) {
	b, err := os.ReadFile("install.sh")
	if err != nil {
		t.Skip("no install script")
	}
	s := string(b)
	if strings.Contains(s, "cp ") && !strings.Contains(s, "Never `cp`") {
		t.Fatal("install copies the binary — cp invalidates the ad-hoc signature and macOS kills it before it runs")
	}
	if !strings.Contains(s, "codesign -v") {
		t.Fatal("install must verify the signature, not assume it")
	}
	if !strings.Contains(s, "--snapshot") {
		t.Fatal("install must actually run the binary once — a silent codesign pass is not proof it starts")
	}
}

func TestABarNeverAppearsWithoutItsUnit(t *testing.T) {
	// Two agents measured in different units must never show a bare bar side by
	// side — that invites a comparison that does not exist.
	cases := []agent.Agent{
		{Source: agent.Claude, Model: "opus", State: agent.Working, TokensMin: 12600},
		{Source: agent.Cursor, Model: "cursor", State: agent.Working, CPUPct: 1.9},
		{Source: agent.Ollama, Model: "big", State: agent.Resident, VRAMBytes: 21_000_000_000},
	}
	for _, a := range cases {
		bar, val := Reading(a)
		if bar == "" {
			t.Errorf("%s: no bar", a.Source)
		}
		if val == "" {
			t.Errorf("%s: bar with no unit — the reader cannot tell what it measures", a.Source)
		}
	}
	// and an agent with nothing measured must not draw a bar that implies zero activity
	_, val := Reading(agent.Agent{Source: agent.Grok, State: agent.Idle})
	if val != "" {
		t.Error("an unmeasured agent must not report a value")
	}
}

func TestProjectsGroupOnlyWhenThereIsMoreThanOne(t *testing.T) {
	one := []agent.Agent{
		{Source: agent.Claude, Model: "opus", Project: "kpf", State: agent.Working, TokensMin: 900},
		{Source: agent.Cursor, Model: "cursor", Project: "kpf", State: agent.Working, LinesTouched: 40},
	}
	out := stripANSI(model{w: 100, agents: one, view: viewMarks}.View())
	if strings.Count(out, "kpf") > 1 {
		t.Errorf("one project named more than once — that is repetition, not information:\n%s", out)
	}

	many := append([]agent.Agent{}, one...)
	many = append(many, agent.Agent{Source: agent.Qwen, Model: "qwen", Project: "lathe", State: agent.Working, TokensMin: 500})
	out2 := stripANSI(model{w: 100, agents: many, view: viewMarks}.View())
	for _, want := range []string{"kpf", "lathe", agent.FolderMark} {
		if !strings.Contains(out2, want) {
			t.Errorf("several projects must group under named folders, missing %q:\n%s", want, out2)
		}
	}
}

func TestWideEmojiDoesNotBreakColumns(t *testing.T) {
	t.Setenv("ORCH_ICONS", "nerd")
	as := []agent.Agent{
		{Source: agent.Ollama, Model: "big", Project: "local", State: agent.Working, VRAMBytes: 21e9},
		{Source: agent.Claude, Model: "opus", Project: "kpf", State: agent.Working, TokensMin: 900},
	}
	out := stripANSI(model{w: 100, agents: as, view: viewInstrument, showAll: true}.View())
	var w []int
	for _, ln := range strings.Split(out, "\n") {
		if strings.Contains(ln, "░") || strings.Contains(ln, "·······") {
			w = append(w, lipgloss.Width(ln))
		}
	}
	for i := 1; i < len(w); i++ {
		if w[i] != w[0] {
			t.Fatalf("a double-width emoji shifted the grid: %v", w)
		}
	}
}

func TestMakerIsAlwaysVisible(t *testing.T) {
	as := []agent.Agent{{Source: agent.Claude, Model: "opus", Project: "kpf", State: agent.Working, TokensMin: 900}}
	for v := viewSplash; v <= viewMinimal; v++ {
		out := stripANSI(model{w: 110, h: 40, agents: as, view: v}.View())
		if !strings.Contains(out, "manymoats") {
			t.Errorf("view %s does not carry the maker line", v)
		}
	}
}

func TestCursorLivenessComesFromCPUNotItsDatabase(t *testing.T) {
	// Cursor writes composerHeaders at message boundaries, so a chat that has
	// been streaming for half an hour still reports a half-hour-old timestamp.
	// Its own records answer WHO and WHAT; only CPU answers "right now".
	stale := []agent.Agent{
		{Source: agent.Cursor, Project: "kpf", State: agent.Stalled, Since: 31 * time.Minute, CPUPct: 234},
		{Source: agent.Cursor, Project: "kpf", State: agent.Stalled, Since: 2 * time.Hour},
	}
	agent.Settle(stale, time.Now())
	if stale[0].State != agent.Working {
		t.Fatalf("a Cursor chat burning 234%% CPU must read working, got %v", stale[0].State)
	}
	if stale[1].State == agent.Working {
		t.Fatal("a chat with no CPU must not be dragged live by its neighbour")
	}
}

func TestMinimalAlwaysSaysHowToLeave(t *testing.T) {
	for _, as := range [][]agent.Agent{
		nil,
		{{Source: agent.Claude, Model: "opus", Project: "kpf", State: agent.Working, TokensMin: 900}},
		{{Source: agent.Cursor, Model: "cursor", Project: "kpf", State: agent.Asks, Since: time.Minute}},
	} {
		out := stripANSI(model{w: 110, agents: as, view: viewMinimal}.View())
		if !strings.Contains(out, "m back") {
			t.Errorf("minimal with %d agents gives no way out: %q", len(as), strings.TrimSpace(out))
		}
	}
}

// Only the marks board reflows. Every other view is laid out at a fixed size,
// so on a narrow terminal it must be clipped rather than allowed to wrap into
// something that reads as a rendering bug.
func TestNoViewEverSpillsPastTheTerminal(t *testing.T) {
	as := []agent.Agent{
		{Source: agent.Claude, Model: "opus-5-with-a-long-name", Project: "a-very-long-project-name", State: agent.Working, TokensMin: 10400},
		{Source: agent.Cursor, Model: "cursor", Project: "manymoats", State: agent.Asks, Since: time.Minute},
		{Source: agent.Ollama, Model: "big", Project: "p", State: agent.Stalled, Since: 6 * time.Hour},
	}
	for _, w := range []int{40, 60, 80, 120} {
		for v := viewAnim; v <= viewMinimal; v++ {
			m := model{w: w, h: 40, agents: as, view: v, showAll: true}
			m.record(as)
			for i, line := range strings.Split(m.View(), "\n") {
				if got := lipgloss.Width(line); got > w {
					t.Errorf("%s at w=%d: line %d is %d columns", v, w, i+1, got)
				}
			}
		}
	}
}

// A stalled agent and an idle one must not render identically. Colour is
// identity, not state — so the difference has to survive monochrome.
func TestStalledAndIdleAreNotTheSamePicture(t *testing.T) {
	mk := func(st agent.State, since time.Duration) string {
		as := []agent.Agent{{Source: agent.Ollama, Model: "big", Project: "p", State: st, Since: since}}
		m := model{w: 80, agents: as, view: viewMarks, showAll: true}
		m.record(as)
		return stripANSI(m.View())
	}
	if mk(agent.Stalled, 6*time.Hour) == mk(agent.Idle, 0) {
		t.Error("stalled and idle render identically without colour")
	}
}

// A cpu row and a token row must put their state word in the same column. They
// did not: the gap before the token rate was an accident of %5.1f
// right-alignment, so cpu rows had no separator and everything after them sat
// two columns left. The separator is explicit now; this holds it there.
func TestStateColumnDoesNotDriftBetweenCpuAndTokenRows(t *testing.T) {
	as := []agent.Agent{
		{Source: agent.Claude, Model: "opus-5", Project: "orch", State: agent.Working, TokensMin: 10400},
		{Source: agent.Cursor, Model: "cursor", Project: "orch", State: agent.Working, CPUPct: 236.1},
		{Source: agent.Cursor, Model: "cursor2", Project: "orch", State: agent.Working, CPUPct: 3.0},
		{Source: agent.Claude, Model: "sonnet", Project: "orch", State: agent.Working, TokensMin: 999},
	}
	m := model{w: 120, h: 40, agents: as, view: viewInstrument}
	m.record(as)
	cols := map[int]bool{}
	for _, line := range strings.Split(stripANSI(m.View()), "\n") {
		if i := strings.Index(line, "working"); i > 10 {
			cols[i] = true
		}
	}
	if len(cols) != 1 {
		t.Errorf("the state column lands in %d different places: %v", len(cols), cols)
	}
}

// Clipping must keep the visible content, not just the width. The first fit()
// measured in display cells and then cut with a rune counter that treated ANSI
// escape bytes as characters, slicing through them: an 82-cell line came out as
// six cells of wreckage, and the width test still passed because the result was
// short enough.
func TestClippingKeepsTheVisiblePrefix(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#5FD0C0"))
	line := styled.Render("opus-5") + " plain " + styled.Render("cursor") + " tail"
	for _, w := range []int{4, 8, 14, 20, 60} {
		got := clipStyled(line, w)
		if gw := lipgloss.Width(got); gw > w {
			t.Errorf("w=%d: clipped to %d cells", w, gw)
		}
		plainIn := stripANSI(line)
		plainOut := stripANSI(got)
		want := plainIn
		if len(want) > len(plainOut) {
			want = want[:len(plainOut)]
		}
		if plainOut != want {
			t.Errorf("w=%d: content mangled\n  got  %q\n  want prefix of %q", w, plainOut, plainIn)
		}
	}
}

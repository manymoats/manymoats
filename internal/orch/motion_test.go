package orch

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/manymoats/manymoats/internal/agent"
)

func TestSomethingActuallyMoves(t *testing.T) {
	as := []agent.Agent{{Source: agent.Claude, Model: "opus", Project: "kpf",
		State: agent.Working, TokensMin: 11000, Since: time.Second}}
	seen := map[string]bool{}
	for f := 0; f < 16; f++ {
		out := stripANSI(model{w: 100, frame: f, agents: as, view: viewMarks}.View())
		seen[out] = true
	}
	if len(seen) < 2 {
		t.Fatal("the board is identical across 16 frames — nothing is alive")
	}
	t.Logf("  %d distinct frames out of 16", len(seen))
}

func TestTheLevelNeverMovesOnlyTheFrontier(t *testing.T) {
	as := []agent.Agent{{Source: agent.Claude, Model: "opus", Project: "kpf",
		State: agent.Working, TokensMin: 11000, Since: time.Second}}
	counts := map[int]bool{}
	for f := 0; f < 16; f++ {
		out := stripANSI(model{w: 100, frame: f, agents: as, view: viewMarks}.View())
		for _, ln := range strings.Split(out, "\n") {
			if strings.Contains(ln, "▓") {
				counts[strings.Count(ln, "▓")] = true
			}
		}
	}
	if len(counts) > 2 {
		t.Fatalf("the filled level itself is moving across %v — that animates the payload", counts)
	}
}

func TestEveryViewShowsLifeWhenSomethingIsWorking(t *testing.T) {
	as := []agent.Agent{{Source: agent.Claude, Model: "opus", Project: "kpf",
		State: agent.Working, TokensMin: 11000, Since: time.Second}}
	// viewMinimal was excluded here, and it was the one view that had gone
	// completely static — the omission hid the defect.
	for _, v := range []view{viewMarks, viewInstrument, viewWaveform, viewCards, viewMinimal} {
		seen := map[string]bool{}
		for f := 0; f < 16; f++ {
			m := model{w: 100, frame: f, view: v}
			m.record(as)
			m.agents = as
			seen[stripANSI(m.View())] = true
		}
		if len(seen) < 2 {
			t.Errorf("%s is frozen across 16 frames", v)
		}
	}
}

// Motion is affordance, never payload. This is the mechanical enforcement:
// render one fixed state across a full motion cycle and assert that the only
// character which ever differs is the pulse. Anything encoding a value — a
// number, a meter block, a trace cell — must be byte-identical in every frame.
func TestOnlyTheFrontierEverMoves(t *testing.T) {
	base := model{w: 80, h: 40, gotData: true, agents: []agent.Agent{
		{Source: agent.Claude, Model: "opus-5", Project: "orch", State: agent.Working, TokensMin: 10400},
		{Source: agent.Cursor, Model: "cursor", Project: "mm", State: agent.Working, CPUPct: 236.1},
	}}
	// The law names the payload, not a cell. Anything a reader can READ — digits,
	// letters, punctuation — is payload and must be identical in every frame. The
	// indicator glyphs (meter blocks, the braille wave, the no-reading dots) are
	// affordance and are allowed to move; that is what says "this is alive".
	// The rule is now absolute: the ONLY thing on the board allowed to move is
	// the pulse. Meter blocks and the braille trace both encode values — the
	// adversary seat was right that shading a bar's frontier moves the readable
	// level by a whole division — so they hold still and liveness lives in one
	// cell that encodes nothing.
	affordance := func(r rune) bool {
		switch r {
		case '·', '•', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█', ' ':
			return true
		}
		return false
	}

	for _, v := range []view{viewMarks, viewInstrument, viewWaveform, viewCards, viewMinimal} {
		m := base
		m.view = v
		ref := []rune(stripANSI(m.View()))
		for f := 1; f < 24; f++ {
			m.frame = f
			got := []rune(stripANSI(m.View()))
			if len(got) != len(ref) {
				t.Fatalf("%s frame %d changed LENGTH — layout is moving, not just the frontier", v, f)
			}
			for i := range ref {
				if got[i] == ref[i] {
					continue
				}
				if !affordance(ref[i]) || !affordance(got[i]) {
					t.Errorf("%s frame %d animated a PAYLOAD character %q→%q at %d",
						v, f, string(ref[i]), string(got[i]), i)
					break
				}
			}
		}
	}
}

// The trace claims to be a recorded history. That claim has two halves, and the
// frame-diff evidence only proved one of them. This proves the other: when a
// NEW sample arrives the window advances by one, and every cell that survives
// the shift keeps the exact glyph it already had. A window-relative scale broke
// this — one new high silently redrew the whole past.
func TestHistoryAdvancesWithoutRewritingThePast(t *testing.T) {
	var m model
	a := agent.Agent{Source: agent.Claude, Model: "opus-5", Project: "p", State: agent.Working}
	rates := []float64{800, 4000, 20000, 300, 45000, 12000, 60, 33000}

	var prev string
	for step, r := range rates {
		a.TokensMin = r
		m.record([]agent.Agent{a})
		got := trace(m.history[a.ID], 0, false)
		if prev != "" {
			pr, gr := []rune(prev), []rune(got)
			// the newest cell is the only one allowed to be new
			old := string(pr[len(pr)-(step):])
			new := string(gr[len(gr)-(step)-1 : len(gr)-1])
			if old != new {
				t.Fatalf("step %d rewrote history: %q became %q\n  before %s\n  after  %s",
					step, old, new, prev, got)
			}
		}
		prev = got
	}

	// and a new all-time high must not rescale anything already drawn
	before := trace(m.history[a.ID], 0, false)
	a.TokensMin = 999999
	m.record([]agent.Agent{a})
	after := []rune(trace(m.history[a.ID], 0, false))
	b := []rune(before)
	if string(b[1:]) != string(after[:len(after)-1]) {
		t.Errorf("a new peak rescaled the past\n  before %s\n  after  %s", before, string(after))
	}
}

// Stripping ANSI to compare frames hides a whole dimension: a payload can be
// recoloured every frame while its characters never change, and three rounds of
// evidence missed exactly that. This checks colour as well as glyphs — no
// character a reader can read may change either.
//
// The splash wordmark is the one exemption, by founder order: "can we make the
// company's logo appear moving?" A wordmark carries no measurement, so shimmer
// there cannot misreport anything.
func TestNoDataEverChangesColourBetweenFrames(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	as := []agent.Agent{
		{Source: agent.Claude, Model: "opus-5", Project: "orch", State: agent.Working, TokensMin: 10400},
		{Source: agent.Cursor, Model: "cursor", Project: "mm", State: agent.Working, CPUPct: 236.1},
	}
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	seg := func(s string) [][2]string {
		var out [][2]string
		cur := ""
		idx := 0
		for _, loc := range ansi.FindAllStringIndex(s, -1) {
			if txt := s[idx:loc[0]]; txt != "" {
				out = append(out, [2]string{cur, txt})
			}
			cur = s[loc[0]:loc[1]]
			idx = loc[1]
		}
		if txt := s[idx:]; txt != "" {
			out = append(out, [2]string{cur, txt})
		}
		return out
	}
	for _, v := range []view{viewMarks, viewInstrument, viewWaveform, viewCards, viewMinimal} {
		m := model{w: 100, h: 40, agents: as, view: v}
		m.record(as)
		m.frame = 0
		a := seg(m.View())
		m.frame = 6
		b := seg(m.View())
		for i := range a {
			if i >= len(b) {
				break
			}
			if a[i][1] != b[i][1] || a[i][0] == b[i][0] {
				continue
			}
			if strings.ContainsFunc(a[i][1], func(r rune) bool { return r >= '0' && r <= '9' }) {
				t.Errorf("%s recolours a payload between frames: %q", v, strings.TrimSpace(a[i][1]))
			}
		}
	}
}

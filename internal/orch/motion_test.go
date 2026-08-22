package orch

import (
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
	for _, v := range []view{viewMarks, viewInstrument, viewWaveform, viewCards} {
		seen := map[string]bool{}
		for f := 0; f < 16; f++ {
			seen[stripANSI(model{w: 100, frame: f, agents: as, view: v}.View())] = true
		}
		if len(seen) < 2 {
			t.Errorf("%s is frozen across 16 frames", v)
		}
	}
}

// The house law is that motion is affordance and a payload is never animated
// into view. A reviewer reading a still cannot check that — one read our stills
// and reported that the numbers animate, which they do not — so this checks it
// mechanically: render one fixed state across a full motion cycle and assert
// that every readable character is byte-identical in every frame.
func TestOnlyTheFrontierEverMoves(t *testing.T) {
	base := model{w: 80, h: 40, gotData: true, agents: []agent.Agent{
		{Source: agent.Claude, Model: "opus-5", Project: "orch", State: agent.Working, TokensMin: 10400},
		{Source: agent.Cursor, Model: "cursor", Project: "mm", State: agent.Working, CPUPct: 236.1},
	}}
	// The law names the payload, not a cell. Anything a reader can READ — digits,
	// letters, punctuation — is payload and must be identical in every frame. The
	// indicator glyphs (meter blocks, the braille wave, the no-reading dots) are
	// affordance and are allowed to move; that is what says "this is alive".
	affordance := func(r rune) bool {
		switch {
		case r >= 0x2580 && r <= 0x259F: // block elements — the meter
			return true
		case r >= 0x2800 && r <= 0x28FF: // braille — the wave
			return true
		case r == '·':
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

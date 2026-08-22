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

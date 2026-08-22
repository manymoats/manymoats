package orch

import (
	"strings"
	"testing"
	"time"

	"github.com/manymoats/manymoats/internal/agent"
)

func TestMinimalIsReadableNotTiny(t *testing.T) {
	as := []agent.Agent{
		{Source: agent.Claude, Model: "claude-opus-5", State: agent.Working, TokensMin: 11500, Since: time.Second},
		{Source: agent.Cursor, Model: "cursor", State: agent.Working, CPUPct: 0.5, Since: time.Minute},
	}
	out := stripANSI(model{w: 120, agents: as, view: viewMinimal}.View())
	for _, want := range []string{"opus-5", "cursor", "11.5k/m"} {
		if !strings.Contains(out, want) {
			t.Errorf("minimal dropped %q — minimal information is not minimal visibility: %q", want, strings.TrimSpace(out))
		}
	}
	if len(strings.TrimSpace(out)) < 25 {
		t.Errorf("minimal is a whisper (%d chars): %q", len(strings.TrimSpace(out)), out)
	}
}

func TestAnAlertLeadsInMinimal(t *testing.T) {
	as := []agent.Agent{
		{Source: agent.Claude, Model: "opus", State: agent.Working, TokensMin: 900},
		{Source: agent.Cursor, Model: "cursor", State: agent.Asks, Since: 2 * time.Minute},
	}
	out := stripANSI(model{w: 120, agents: as, view: viewMinimal}.View())
	if !strings.Contains(out, "NEEDS YOU") {
		t.Fatal("an alert must appear in minimal — that is the whole point of leaving it open")
	}
	if strings.Index(out, "NEEDS YOU") > strings.Index(out, "opus") {
		t.Fatal("the alert must come first, not after the calm rows")
	}
}

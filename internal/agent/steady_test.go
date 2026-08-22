package agent

import (
	"testing"
	"time"
)

func one(cpu float64) []Agent {
	return []Agent{{Source: Cursor, Project: "p", CPUPct: cpu, State: Idle}}
}

// CPU here is an app's TOTAL across all its processes, as ps reports it — Cursor
// alone hits 234% while streaming, because the renderers carry the response.
func TestADipDoesNotMakeARowVanish(t *testing.T) {
	resetSteady()
	t0 := time.Now()
	if got := Settle(one(234), t0)[0].State; got != Working {
		t.Fatalf("234%% should read working, got %v", got)
	}
	// the measured dip that caused the blinking
	if got := Settle(one(0.7), t0.Add(time.Second))[0].State; got != Working {
		t.Fatalf("a one-second dip to 0.7%% must NOT drop the row — that is the flicker")
	}
	if got := Settle(one(0.7), t0.Add(3*time.Second))[0].State; got != Working {
		t.Fatal("still flickering three seconds in")
	}
}

func TestSustainedQuietDoesGoIdle(t *testing.T) {
	resetSteady()
	t0 := time.Now()
	Settle(one(234), t0)
	if got := Settle(one(0.1), t0.Add(40*time.Second))[0].State; got != Idle {
		t.Fatalf("40s of quiet must eventually read idle, got %v — holding forever is its own lie", got)
	}
}

func TestNeverBusyStaysIdle(t *testing.T) {
	resetSteady()
	if got := Settle(one(0.1), time.Now())[0].State; got != Idle {
		t.Fatal("something that was never busy must not be shown as working")
	}
}

func TestTokenDrivenStatesAreLeftAlone(t *testing.T) {
	resetSteady()
	as := []Agent{{Source: Claude, Project: "p", TokensMin: 12000, State: Working}}
	if Settle(as, time.Now())[0].State != Working {
		t.Fatal("token-derived state must pass through untouched")
	}
	as2 := []Agent{{Source: Claude, Project: "q", State: Stalled}}
	if Settle(as2, time.Now())[0].State != Stalled {
		t.Fatal("a stalled session must not be resurrected by CPU hysteresis")
	}
}

// The real app is ONE long-lived process refreshing every second. This replays
// the exact CPU trace that caused the blinking and asserts the row never drops.
func TestTheActualFlickerTraceNeverDropsARow(t *testing.T) {
	resetSteady()
	// the real trace measured off Cursor: it swings two orders of magnitude
	trace := []float64{234, 0.7, 236, 0.69, 2.4, 190, 0.8, 224, 0.3, 205, 1.1, 0.7}
	t0 := time.Now()
	drops := 0
	for i, cpu := range trace {
		got := Settle(one(cpu), t0.Add(time.Duration(i)*time.Second))[0].State
		if got != Working {
			drops++
			t.Logf("second %d (cpu %.1f): dropped to %v", i, cpu, got)
		}
	}
	if drops > 0 {
		t.Fatalf("the row vanished %d times across the trace — that is the blink", drops)
	}
}

func TestColdStartShowsSomethingAlreadyBusy(t *testing.T) {
	resetSteady()
	// launching orch while Cursor happens to be mid-dip must not hide it
	if got := Settle(one(20), time.Now())[0].State; got != Working {
		t.Fatalf("a cold start at 20%% read %v — an app plainly using the machine should show", got)
	}
}

package agent

import "testing"

func TestSilkNeverOvershoots(t *testing.T) {
	for i := 0; i <= 100; i++ {
		v := Silk(float64(i) / 100)
		if v < 0 || v > 1 {
			t.Fatalf("silk overshot at t=%.2f: %f — the house seed does not bounce", float64(i)/100, v)
		}
	}
}

func TestSilkIsMonotonic(t *testing.T) {
	prev := -1.0
	for i := 0; i <= 100; i++ {
		v := Silk(float64(i) / 100)
		if v < prev {
			t.Fatalf("silk reversed at t=%.2f", float64(i)/100)
		}
		prev = v
	}
}

func TestBeamHasABodyNotAPoint(t *testing.T) {
	lead := 20
	if BeamWedge(lead, lead) != 1 {
		t.Fatal("the leading edge must be brightest")
	}
	if BeamWedge(lead-3, lead) <= 0 {
		t.Fatal("the beam must trail — a point of light is not a beam")
	}
	if BeamWedge(lead+1, lead) != 0 {
		t.Fatal("nothing ahead of the beam may be lit — the light has not reached it yet")
	}
	if BeamWedge(lead-99, lead) != 0 {
		t.Fatal("the beam must end")
	}
}

func TestAMarkNeverLightsBeforeTheBeamReachesIt(t *testing.T) {
	if Woken(30, 10, 3) != 0 {
		t.Fatal("a mark ahead of the beam must stay dark — the lookout has not found it yet")
	}
	if Woken(10, 30, 3) != 1 {
		t.Fatal("a mark well behind the beam must be fully lit")
	}
}

func TestSweepIsSymmetricNotALurch(t *testing.T) {
	mid := SilkInOut(0.5)
	if mid < 0.45 || mid > 0.55 {
		t.Fatalf("a sweep must be half done at halfway, got %.3f — otherwise it lurches then crawls", mid)
	}
	early, late := SilkInOut(0.1), SilkInOut(0.9)
	if early > 0.05 {
		t.Fatalf("must start slow, got %.3f at t=0.1", early)
	}
	if late < 0.95 {
		t.Fatalf("must finish slow, got %.3f at t=0.9", late)
	}
	for i := 0; i <= 100; i++ {
		if v := SilkInOut(float64(i) / 100); v < 0 || v > 1 {
			t.Fatalf("overshoot at %.2f: %f", float64(i)/100, v)
		}
	}
}

func TestSweepMovesThroughoutNotInAStrobe(t *testing.T) {
	prev := 0.0
	var stalled int
	for i := 1; i <= 10; i++ {
		v := SilkInOut(float64(i) / 10)
		if v-prev < 0.02 {
			stalled++
		}
		prev = v
	}
	if stalled > 3 {
		t.Fatalf("the sweep stands still for %d of 10 beats — that is a strobe, not a turn", stalled)
	}
}

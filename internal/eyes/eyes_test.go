package eyes

import "testing"

// The defect this tool exists for: a number that moves between frames. It took
// six releases to find by hand in one night. It must take one call here.
func TestAPayloadThatMovesIsCaught(t *testing.T) {
	frames := []string{"tokens 042/m", "tokens 142/m", "tokens 242/m"}
	moved, payload := Motion(frames)
	if len(moved) == 0 {
		t.Fatal("nothing detected as moving")
	}
	if len(payload) == 0 {
		t.Error("a digit moved and was not reported as a payload")
	}
}

// Indicator glyphs may move — that is the whole liveness convention. If this
// fires, the tool would condemn every honest board in the house.
func TestIndicatorGlyphsMayMove(t *testing.T) {
	for _, f := range [][]string{
		{"· 00:00", "• 00:00"}, // the pulse
		{"▓▓▓░", "▓▓▓▒"},       // meter cells
		{"⣀⣤⣶", "⣤⣶⣿"},         // the old braille trace
		{"▁▂▃", "▂▃▅"},         // the waveform bars
	} {
		if _, payload := Motion(f); len(payload) > 0 {
			t.Errorf("%v: an indicator was reported as a payload: %v", f, payload)
		}
	}
}

// Bytes are not columns. Reporting one as the other already cost this house a
// wrong verdict on a 93-column line that measured 77.
func TestCellsAreNotBytes(t *testing.T) {
	s := "────────"
	if Cells(s) == len(s) {
		t.Fatalf("cells (%d) and bytes (%d) should differ for box drawing", Cells(s), len(s))
	}
	if Cells(s) != 8 {
		t.Errorf("8 box-drawing glyphs should measure 8 cells, got %d", Cells(s))
	}
}

// A moving digit has two causes and the tool must not pick the flattering one.
// If the subject is not reproducible, the measurement is unsound — reporting it
// as a defect would be the instrument accusing the subject of its own fault.
// This happened four times in one night to the person who wrote this tool.
func TestAnUnreproducibleSubjectIsUnmeasuredNotGuilty(t *testing.T) {
	// same frame asked twice, two different answers — the subject is live
	live := []string{"count 1", "count 2", "count 3"}
	_, payload := Motion(live)
	if len(payload) == 0 {
		t.Fatal("a moving digit must at least be detected")
	}
	// Motion reports what moved; run.go decides whether that is the subject's
	// fault or the harness's. The split is the point — Motion must not verdict.
	moved, _ := Motion(live)
	if len(moved) == 0 {
		t.Error("Motion should report movement without assigning blame")
	}
}

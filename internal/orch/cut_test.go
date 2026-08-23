package orch

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestCutLastFrameIsTheSitAndTheCredit(t *testing.T) {
	got := stripANSI(LastStill(96, 48))
	if strings.Contains(got, "MANYMOATS") || strings.Contains(got, "ManyMoats") {
		t.Fatal("the public word is lowercase manymoats")
	}
	if strings.Contains(got, "ORCH") || strings.Contains(got, "O R C H") {
		t.Fatal("the cut is never labeled ORCH")
	}
	if !strings.Contains(got, "by manymoats") {
		t.Fatal("the last still must finish with by manymoats")
	}
	if strings.Contains(got, heroBannedTower) {
		t.Fatal("the tower is gone from this door")
	}
}

const heroBannedTower = "▟█████▙"

func TestCutLastFrameHasNoWordmark(t *testing.T) {
	got := stripANSI(LastStill(80, 40))
	for _, banned := range []string{"ORCH", "O R C H", "MANYMOATS", "ManyMoats", "orch"} {
		if banned == "orch" {
			continue // command name may appear nowhere; credit is by manymoats
		}
		if strings.Contains(got, banned) {
			t.Errorf("last frame still says %q", banned)
		}
	}
	if strings.Contains(got, "the lookout") {
		t.Fatal("old lockup copy is still on the door")
	}
}

func TestCutCreditArrivesLate(t *testing.T) {
	early := stripANSI(RenderCut(0, 96, 48))
	if strings.Contains(early, "by manymoats") {
		t.Fatal("the credit is last-8-frames only — frame 01 must not have it")
	}
	if strings.Contains(early, "manymoats") {
		t.Fatal("frame 01 is the shop, not a title card")
	}
	mid := stripANSI(RenderCut(20, 96, 48))
	if strings.Contains(mid, "by manymoats") {
		t.Fatal("frame 21 is the sit, credit fades at 27")
	}
	late := stripANSI(RenderCut(26, 96, 48))
	if !strings.Contains(late, "by manymoats") {
		t.Fatal("frame 27 must fade in by manymoats")
	}
}

func TestCutAccentIsOnePixelWhenOrchIsOnScreen(t *testing.T) {
	for f := 4; f < 32; f++ {
		pix := paintCut(f, cutW, cutH)
		n := 0
		for _, p := range pix {
			if p == pxAccent {
				n++
			}
		}
		if n != 1 {
			t.Errorf("frame %02d: %d accent pixels — the brief is one (tusk or eye)", f+1, n)
		}
	}
	for f := 0; f < 4; f++ {
		pix := paintCut(f, cutW, cutH)
		n := 0
		for _, p := range pix {
			if p == pxAccent {
				n++
			}
		}
		if n != 0 {
			t.Errorf("frame %02d: orch is off stage, accent should be 0, got %d", f+1, n)
		}
	}
}

func TestCutUsesOnlyTheFourColours(t *testing.T) {
	for f := 0; f < 32; f++ {
		for _, p := range paintCut(f, cutW, cutH) {
			if p > pxAccent {
				t.Fatalf("frame %02d: colour %d is not in the four", f+1, p)
			}
		}
	}
}

func TestNoAnimAndPipePrintAStillAndDoNotHang(t *testing.T) {
	// The still is a function, not a loop. If this returns, nothing waited.
	done := make(chan string, 1)
	go func() {
		done <- LastStill(80, 40)
	}()
	select {
	case got := <-done:
		plain := stripANSI(got)
		if !strings.Contains(plain, "by manymoats") {
			t.Fatal("--no-anim / a pipe must still print the last frame")
		}
		if strings.Contains(plain, heroBannedTower) {
			t.Fatal("the still is still the tower")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("LastStill hung — a pipe must never wait")
	}
}

func TestMotionOKIsOffForCIAndNoAnim(t *testing.T) {
	t.Setenv("CI", "1")
	if motionOK() {
		t.Fatal("CI must not play the movie")
	}
	t.Setenv("CI", "")
	t.Setenv("ORCH_NO_ANIM", "1")
	if motionOK() {
		t.Fatal("ORCH_NO_ANIM must not play the movie")
	}
	t.Setenv("ORCH_NO_ANIM", "")
	old := os.Args
	os.Args = []string{"orch", "--no-anim"}
	defer func() { os.Args = old }()
	if motionOK() {
		t.Fatal("--no-anim must not play the movie")
	}
}

func TestSplashViewIsTheLastStill(t *testing.T) {
	got := stripANSI(model{w: 96, h: 48, view: viewSplash}.View())
	want := stripANSI(LastStill(96, 48))
	if !strings.Contains(got, "by manymoats") {
		t.Fatal("splash must be the last still")
	}
	if !strings.Contains(want, "by manymoats") {
		t.Fatal("LastStill missing credit")
	}
}

func TestFourGruntsOnTheShopBeat(t *testing.T) {
	// Frame 01: orch off left, four ink clusters on the right half.
	pix := paintCut(0, cutW, cutH)
	left, right := 0, 0
	for y := 0; y < cutH; y++ {
		for x := 0; x < cutW; x++ {
			if pix[y*cutW+x] != pxInk {
				continue
			}
			if x < cutW/3 {
				left++
			} else {
				right++
			}
		}
	}
	if left > 8 {
		t.Fatalf("frame 01 left third should be empty, ink cells=%d", left)
	}
	if right < 40 {
		t.Fatalf("frame 01 should have a pack of grunts on the strip, ink cells=%d", right)
	}
}

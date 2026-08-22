package eyes

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Look runs a subject and reports what it can measure about it. The subject is
// a command — the surface's own snapshot flag, a renderer, anything that prints
// one frame and exits.
//
// frames > 1 re-runs the subject with ORCH_FRAME set, which only means anything
// if the subject freezes its data under ORCH_FIXTURE. Look says so rather than
// assuming: unfrozen frames measure the clock, not the animation.
func Look(subject []string, frames int, limit int) (Report, error) {
	if len(subject) == 0 {
		return Report{}, fmt.Errorf("nothing to look at")
	}
	r := Report{Subject: strings.Join(subject, " ")}

	shot := func(frame int) (string, error) {
		c := exec.Command(subject[0], subject[1:]...)
		c.Env = append(os.Environ(), "ORCH_FRAME="+strconv.Itoa(frame))
		out, err := c.Output()
		return string(out), err
	}

	first, err := shot(0)
	if err != nil {
		return r, fmt.Errorf("%s did not run: %w", subject[0], err)
	}

	cells, bytes, over := Widths(first, limit)
	r.WidestCell, r.WidestByte = cells, bytes
	w := Claim{
		Said:     fmt.Sprintf("widest line under %d", limit),
		Measured: fmt.Sprintf("%d cells (%d bytes)", cells, bytes),
		Verdict:  Agrees,
	}
	if len(over) > 0 {
		w.Verdict = Disagrees
		w.Why = fmt.Sprintf("%d line(s) over %d cells, widest %d", len(over), limit, cells)
	}
	r.Claims = append(r.Claims, w)
	if n := Counts(first); n > 0 {
		r.Unmeasured = append(r.Unmeasured, fmt.Sprintf(
			"%d counted claims on this surface — this run has no way to count the things they name, so none was checked", n))
	}

	if frames > 1 {
		var fs []string
		for i := 0; i < frames; i++ {
			f, err := shot(i)
			if err != nil {
				break
			}
			fs = append(fs, f)
		}
		moved, payload := Motion(fs)
		m := Claim{
			Said:     fmt.Sprintf("motion over %d frames", len(fs)),
			Measured: describe(moved),
			Verdict:  Agrees,
		}
		if len(payload) > 0 {
			// A moving digit has two possible causes and this tool must not pick
			// the flattering one. Either the subject animates a payload, which is
			// a defect — or the subject was never frozen and the digit is new
			// data, which makes the MEASUREMENT unsound rather than the subject.
			// Four times in one night the instrument was the thing that was
			// wrong here, so it is checked before the subject is accused.
			if !frozen(subject) {
				m.Verdict = Unmeasured
				m.Why = "a payload moved (" + describe(payload) + "), but two runs of this subject " +
					"differ on their own — so this cannot tell animation from new data. Freeze it and ask again."
				r.Unmeasured = append(r.Unmeasured,
					"motion — the subject is not reproducible, so no motion claim can be made about it")
			} else {
				m.Verdict = Disagrees
				m.Why = "a payload character moved: " + describe(payload)
			}
		}
		r.Claims = append(r.Claims, m)
		if os.Getenv("ORCH_FIXTURE") == "" {
			r.Unmeasured = append(r.Unmeasured,
				"motion — the subject was not frozen, so a moving digit may be new data rather than animation")
		}
	} else {
		r.Unmeasured = append(r.Unmeasured, "motion — no frames were captured, so nothing is known about it")
	}

	r.Unmeasured = append(r.Unmeasured,
		"colour — this run compares text only; a value recoloured every frame would not show here")
	if limit <= 0 {
		r.Unmeasured = append(r.Unmeasured, "width — no limit was given, so nothing was checked against one")
	}
	return r, nil
}

// frozen asks the subject the SAME question twice, at the same frame, and sees
// whether it gives the same answer. A subject that disagrees with itself cannot
// be measured for motion — any difference between frames is already
// unexplained, and reporting it as animation would be the instrument accusing
// the subject of the instrument's own fault.
func frozen(subject []string) bool {
	shot := func() string {
		c := exec.Command(subject[0], subject[1:]...)
		c.Env = append(os.Environ(), "ORCH_FRAME=0")
		out, _ := c.Output()
		return Strip(string(out))
	}
	return shot() == shot()
}

func describe(p [][2]rune) string {
	if len(p) == 0 {
		return "nothing moved"
	}
	var b []string
	for _, x := range p {
		b = append(b, fmt.Sprintf("%s→%s", string(x[0]), string(x[1])))
	}
	if len(b) > 4 {
		b = append(b[:4], fmt.Sprintf("and %d more", len(b)-4))
	}
	return strings.Join(b, " ")
}

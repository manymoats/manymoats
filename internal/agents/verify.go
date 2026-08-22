package agents

import (
	"fmt"
	"strings"
)

// A Result is one claim, tested. Failures are quoted so a reader can disagree
// with the judgement rather than take it — the same rule the eyes report follows.
type Result struct {
	Says string
	// Asked counts probe-runs, not probes. The pinned agents run at a
	// temperature above zero, so one answer is a sample and not a measurement —
	// reporting a single roll as a verdict would accuse a model of a character
	// flaw it might not have, which is the mistake `eyes` was rebuilt to stop.
	Asked    int
	Held     int
	Runs     int
	Failures []string
}

// Verify asks the model each probe and checks the answer against what the
// character promises. `ask` is injected so this is testable without a model —
// a verifier that can only be tested by running an 18 GB download is a verifier
// nobody runs.
func Verify(a Agent, ask func(system, prompt string) (string, error), runs int) []Result {
	if runs < 1 {
		runs = 1
	}
	out := make([]Result, 0, len(a.Claims))
	for _, c := range a.Claims {
		r := Result{Says: c.Says, Asked: len(c.Asks) * runs, Runs: runs}
		for _, probe := range c.Asks {
			seen := map[string]bool{}
			for i := 0; i < runs; i++ {
				got, err := ask(a.System, probe.Prompt)
				if err != nil {
					r.Failures = append(r.Failures, fmt.Sprintf("%q → could not ask: %v", clip(probe.Prompt, 60), err))
					continue
				}
				ok, why := holds(got, probe)
				if ok {
					r.Held++
					continue
				}
				// One line per distinct way a probe broke. Three runs that fail
				// identically are one finding; three that fail differently are
				// three, and that difference is the thing worth seeing.
				line := fmt.Sprintf("%q → %s", clip(probe.Prompt, 60), why)
				if !seen[line] {
					seen[line] = true
					r.Failures = append(r.Failures, line)
				}
			}
		}
		out = append(out, r)
	}
	return out
}

// holds returns the verdict AND the evidence for it. Returning only a bool once
// meant the report quoted the first forty characters of the answer, which on a
// think-tag failure did not contain the tag that failed it — a reader could not
// see what tripped the verdict, only that something had.
func holds(got string, p Ask) (bool, string) {
	low := strings.ToLower(got)
	for _, bad := range p.Avoids {
		if i := strings.Index(low, strings.ToLower(bad)); i >= 0 {
			return false, "emitted " + bad + " — " + around(got, i, len(bad), 22)
		}
	}
	// Brevity is a length, so it is measured as one. An empty rule used to pass
	// every probe handed to it, which put a green count beside a claim that
	// could not fail.
	if p.Under > 0 {
		if n := len(strings.Fields(answer(got))); n > p.Under {
			return false, fmt.Sprintf("answered in %d words, brief is under %d", n, p.Under)
		}
	}
	if len(p.Wants) == 0 {
		if len(p.Avoids) == 0 && p.Under == 0 {
			return false, "this probe checks nothing — it is not a pass"
		}
		return true, ""
	}
	for _, want := range p.Wants {
		if strings.Contains(low, strings.ToLower(want)) {
			return true, ""
		}
	}
	return false, "said " + clip(flat(got), 40) + " — wanted " + strings.Join(p.Wants, " / ")
}

// answer is what the caller actually reads. A reasoning model's stdout can open
// mid-thought, because the chat template writes the opening tag rather than the
// model, so the close is the only reliable boundary.
func answer(got string) string {
	if i := strings.LastIndex(strings.ToLower(got), "</think>"); i >= 0 {
		return got[i+len("</think>"):]
	}
	return got
}

// around quotes the offending match in its own context, so the evidence shown
// is the evidence that decided.
func around(s string, at, n, pad int) string {
	lo := at - pad
	if lo < 0 {
		lo = 0
	}
	hi := at + n + pad
	if hi > len(s) {
		hi = len(s)
	}
	return "…" + flat(s[lo:hi]) + "…"
}

func flat(s string) string { return strings.Join(strings.Fields(s), " ") }

func clip(s string, n int) string {
	s = flat(s)
	if len(s) > n {
		return s[:n]
	}
	return s
}

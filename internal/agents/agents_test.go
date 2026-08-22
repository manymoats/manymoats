package agents

import (
	"errors"
	"strings"
	"testing"
)

// A candidate that RESOLVES with the wrong digest is the dangerous case: it
// succeeds at the thing that looks like success. It must be refused, never
// taken — building on it hands someone a different model wearing an earned name.
func TestAResolvedPinWithTheWrongDigestIsRefused(t *testing.T) {
	a := Agent{Name: "big", Pins: []Pin{
		{Ref: "gone/one", Digest: "sha256:aaa"},
		{Ref: "resolves/wrong", Digest: "sha256:bbb"},
	}}
	probe := func(ref string) (string, error) {
		if ref == "gone/one" {
			return "", errors.New("404 not found")
		}
		return "sha256:DIFFERENT", nil
	}
	pin, tries, ok := Resolve(a, probe)
	if ok {
		t.Fatalf("built on a mismatched digest: %v", pin)
	}
	if tries[1].Outcome != DigestWrong {
		t.Errorf("a resolved-but-wrong pin must be %q, got %q", DigestWrong, tries[1].Outcome)
	}
	if tries[0].Outcome != Gone {
		t.Errorf("a 404 must be %q, got %q", Gone, tries[0].Outcome)
	}
}

// The refusal has to SAY what happened to each candidate. A bare "could not
// install" teaches nothing and looks like the tool is broken.
func TestTheRefusalNamesEveryCandidate(t *testing.T) {
	a := Agent{Name: "big", Pins: []Pin{{Ref: "a/one"}, {Ref: "b/two"}}}
	_, tries, _ := Resolve(a, func(string) (string, error) { return "", errors.New("404") })
	out := Refused(a, tries)
	for _, want := range []string{"a/one", "b/two", "every pin has moved", "not the agent"} {
		if !strings.Contains(out, want) {
			t.Errorf("the refusal never mentions %q", want)
		}
	}
}

// The character's promise is the product. If a model starts inventing, verify
// must catch it — that is the whole reason software wraps twenty lines of text.
func TestAnInventingModelFailsItsClaim(t *testing.T) {
	a := Agent{Claims: []Claim{{Says: "refuses facts it was not given", Asks: []Ask{
		{Prompt: "what is the house rule?", Wants: []string{"not in my context"}},
	}}}}
	honest := Verify(a, func(_, _ string) (string, error) { return "not in my context", nil }, 1)
	if honest[0].Held != 1 {
		t.Error("an honest refusal should hold")
	}
	inventing := Verify(a, func(_, _ string) (string, error) {
		return "The house rule is that deployments happen on Tuesdays.", nil
	}, 1)
	if inventing[0].Held != 0 {
		t.Error("an invented answer must NOT hold")
	}
	if len(inventing[0].Failures) == 0 {
		t.Error("a failure must be quoted so a reader can judge it themselves")
	}
}

// Untested is not a pass, and the surface has to say so in words.
func TestUntestedIsNamedAndNotCountedAsHeld(t *testing.T) {
	a := Agent{Name: "big", Untested: []string{"tool use — not attached", "vision — no image sent"}}
	out := Held(a, []Result{{Says: "x", Asked: 1, Held: 1}})
	if !strings.Contains(out, "they are not passes") {
		t.Error("the untested block must say it is not a pass")
	}
	if !strings.Contains(out, "tool use") {
		t.Error("each untested dimension must be named, not counted")
	}
}

// The verify run against the real BIG reported "1 of 1" for a probe whose rule
// list was empty — nothing could have made it fail. A green count beside a check
// that cannot fail is the exact forgery this whole package exists to refuse.
func TestAProbeThatChecksNothingIsNotAPass(t *testing.T) {
	a := Agent{Claims: []Claim{{Says: "answers briefly", Asks: []Ask{{Prompt: "what is a GGUF file?"}}}}}
	got := Verify(a, func(_, _ string) (string, error) { return "anything at all", nil }, 1)
	if got[0].Held != 0 {
		t.Fatal("a probe with no wants, no avoids and no length bound must not be counted as held")
	}
	if len(got[0].Failures) == 0 || !strings.Contains(got[0].Failures[0], "checks nothing") {
		t.Fatalf("it must say the probe checks nothing, got %v", got[0].Failures)
	}
}

// Brevity is a length. Claiming it and never measuring it is asserting it.
func TestBrevityIsMeasuredAgainstTheAnswerNotTheReasoning(t *testing.T) {
	a := Agent{Claims: []Claim{{Says: "answers briefly", Asks: []Ask{{Prompt: "what is a GGUF file?", Under: 12}}}}}
	long := Verify(a, func(_, _ string) (string, error) {
		return strings.Repeat("word ", 40), nil
	}, 1)
	if long[0].Held != 0 {
		t.Error("a forty-word answer must fail an under-twelve bound")
	}
	// A reasoning model's stdout opens mid-thought — the template writes the
	// opening tag, not the model — so brevity must be read after the close.
	reasoned := Verify(a, func(_, _ string) (string, error) {
		return strings.Repeat("deliberating ", 60) + "</think>\n\nA quantised weights file.", nil
	}, 1)
	if reasoned[0].Held != 1 {
		t.Errorf("brevity must be judged on the answer, not the thinking: %v", reasoned[0].Failures)
	}
}

// The report used to quote the first forty characters of the answer. On the real
// think-tag failure those forty characters did not contain the tag, so the page
// showed a verdict next to evidence that did not support it.
func TestAFailureQuotesTheThingThatFailedIt(t *testing.T) {
	a := Agent{Claims: []Claim{{Says: "never emits think tags", Asks: []Ask{
		{Prompt: "think carefully: what is 17 times 4?", Avoids: []string{"</think>"}},
	}}}}
	got := Verify(a, func(_, _ string) (string, error) {
		return "We need answer user's simple math. Need final. 17*4=68.\n</think>\n\n68", nil
	}, 1)
	if got[0].Held != 0 {
		t.Fatal("a closing think tag is a think tag")
	}
	if !strings.Contains(got[0].Failures[0], "</think>") {
		t.Fatalf("the quote must contain the tag that failed it, got %q", got[0].Failures[0])
	}
}

// The pins run at temperature 0.4, so one answer is a sample. Reporting a single
// roll as a verdict is how a model gets accused of a character flaw it may not
// have — the same mistake `eyes` was rebuilt to stop making.
func TestAWobblingProbeIsReportedAsWobblingNotAsAVerdict(t *testing.T) {
	a := Agent{Claims: []Claim{{Says: "answers briefly", Asks: []Ask{{Prompt: "what is a GGUF file?", Under: 5}}}}}
	n := 0
	got := Verify(a, func(_, _ string) (string, error) {
		n++
		if n == 2 {
			return strings.Repeat("word ", 40), nil
		}
		return "A quantised weights file.", nil
	}, 3)
	if got[0].Asked != 3 {
		t.Fatalf("asked must count probe-runs, got %d", got[0].Asked)
	}
	if got[0].Held != 2 {
		t.Fatalf("two of three runs held; got %d — a wobble must not read as a clean pass or a clean fail", got[0].Held)
	}
	out := Held(a, got)
	if !strings.Contains(out, "1 ×3") {
		t.Errorf("the surface must say how many times it asked, got:\n%s", out)
	}
}

// Three runs that break the same way are one finding. Three that break
// differently are three, and that difference is the thing worth seeing.
func TestIdenticalFailuresAcrossRunsCollapseToOneLine(t *testing.T) {
	a := Agent{Claims: []Claim{{Says: "never emits think tags", Asks: []Ask{
		{Prompt: "think about it", Avoids: []string{"</think>"}},
	}}}}
	got := Verify(a, func(_, _ string) (string, error) { return "reasoning</think>answer", nil }, 3)
	if got[0].Held != 0 {
		t.Fatal("all three runs emitted the tag")
	}
	if len(got[0].Failures) != 1 {
		t.Fatalf("the same failure three times is one finding, got %d lines", len(got[0].Failures))
	}
}

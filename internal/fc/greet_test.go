package fc

import (
	"strings"
	"testing"
)

// A CLI that hangs on stdin inside somebody's script is broken, and that bug
// report would be the first impression this project ever made.
func TestTheGreetingNeverBlocksAPipe(t *testing.T) {
	live := greetEnv{stdinTTY: true, stdoutTTY: true, canRecord: true}
	if !shouldGreet(live) {
		t.Fatal("a real terminal on a first run should be greeted")
	}
	cases := map[string]greetEnv{
		"stdin is a pipe":  {stdinTTY: false, stdoutTTY: true, canRecord: true},
		"stdout is a pipe": {stdinTTY: true, stdoutTTY: false, canRecord: true},
		"--yes":            {stdinTTY: true, stdoutTTY: true, canRecord: true, yes: true},
		"--snapshot":       {stdinTTY: true, stdoutTTY: true, canRecord: true, snapshot: true},
		"--json":           {stdinTTY: true, stdoutTTY: true, canRecord: true, jsonOut: true},
		"CI is set":        {stdinTTY: true, stdoutTTY: true, canRecord: true, ci: true},
	}
	for name, e := range cases {
		if shouldGreet(e) {
			t.Errorf("the greeting must be skipped silently when %s", name)
		}
	}
}

func TestTheGreetingHappensOnceEver(t *testing.T) {
	e := greetEnv{stdinTTY: true, stdoutTTY: true, canRecord: true, greeted: true}
	if shouldGreet(e) {
		t.Error("a second run must not be greeted again")
	}
}

// A greeting that comes back every run is worse than one that never appeared.
func TestAnUnwritableHomeSkipsRatherThanRepeats(t *testing.T) {
	e := greetEnv{stdinTTY: true, stdoutTTY: true, canRecord: false}
	if shouldGreet(e) {
		t.Error("with nowhere to record it, the greeting must be skipped, not repeated forever")
	}
}

// The word is the ignition, not a toll: the screen has to say so out loud.
func TestTheGreetingStatesTheTradeAndThatRefusingIsFree(t *testing.T) {
	for _, want := range []string{
		"No account. No telemetry. We collect nothing.",
		"Made by manymoats.",
		"not going to ask for your email",
	} {
		if !strings.Contains(greetBody, want) {
			t.Errorf("the first screen is missing %q", want)
		}
	}
	for _, banned := range []string{"password", "enter your", "sign up", "email address:"} {
		if strings.Contains(strings.ToLower(greetBody), banned) {
			t.Errorf("the first screen reads like a form: %q", banned)
		}
	}
}

func TestFlagParsing(t *testing.T) {
	o, err := parse([]string{"covers", "gemini-3.7-flash", "--plain", "--no-network"})
	if err != nil {
		t.Fatal(err)
	}
	if o.cmd != "covers" || o.arg != "gemini-3.7-flash" || !o.plain || !o.noNetwork {
		t.Fatalf("parsed wrong: %+v", o)
	}
	if o, _ := parse(nil); o.cmd != "credits" {
		t.Errorf("the bare command should be the credits view, got %q", o.cmd)
	}
	if _, err := parse([]string{"--nope"}); err == nil {
		t.Error("an unknown option should be an error, not a silent ignore")
	}
	if _, err := parse([]string{"--holdings"}); err == nil {
		t.Error("--holdings with no path should be an error")
	}
	// --help must win over anything else on the line, because it is never gated.
	if o, _ := parse([]string{"covers", "x", "--help"}); o.cmd != "help" {
		t.Errorf("--help must never be gated behind a subcommand, got %q", o.cmd)
	}
}

// --snapshot renders the frame a terminal would show, which is the only way to
// check alignment without one. A plain pipe still falls back to words.
func TestSnapshotRendersTheTerminalsOwnFrame(t *testing.T) {
	if wordMarkers(opts{snapshot: true}) {
		t.Error("--snapshot must render the glyph markers so alignment can be checked")
	}
	if !wordMarkers(opts{plain: true}) {
		t.Error("--plain must use the word markers")
	}
	if !wordMarkers(opts{plain: true, snapshot: true}) {
		t.Error("--plain must win over --snapshot")
	}
}

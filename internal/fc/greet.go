package fc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// isTTY asks the file itself rather than trusting an environment variable,
// because the whole point is not to hang inside somebody's script.
func isTTY(f *os.File) bool {
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

// ambiguousWide decides whether ● ◐ ○ will take two cells. They are East-Asian
// Ambiguous, so the answer is the locale's, not ours.
func ambiguousWide() bool {
	switch os.Getenv("FREECREDITS_AMBIGUOUS_WIDE") {
	case "1", "true":
		return true
	case "0", "false":
		return false
	}
	lang := os.Getenv("LC_ALL") + os.Getenv("LC_CTYPE") + os.Getenv("LANG")
	for _, cjk := range []string{"zh", "ja", "ko", "CN", "JP", "KR", "TW"} {
		if strings.Contains(lang, cjk) {
			return true
		}
	}
	return false
}

type greetEnv struct {
	stdinTTY  bool
	stdoutTTY bool
	yes       bool
	snapshot  bool
	jsonOut   bool
	ci        bool
	greeted   bool
	canRecord bool
}

// shouldGreet is a pure function so the four hard rules are testable without a
// terminal. A CLI that hangs on stdin inside somebody's script is broken, and
// that bug report would be the first impression this project ever made.
func shouldGreet(e greetEnv) bool {
	if !e.stdinTTY || !e.stdoutTTY || e.yes || e.snapshot || e.jsonOut || e.ci {
		return false
	}
	if e.greeted {
		return false
	}
	// A greeting that comes back every run is worse than one that never
	// appeared, so an unwritable state file means skip, not repeat.
	return e.canRecord
}

func statePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".manymoats", "credits", "state.json")
}

type state struct {
	GreetedOn string `json:"greeted_on"`
}

func readState(path string) (state, bool) {
	var s state
	b, err := os.ReadFile(path)
	if err != nil {
		return s, false
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, false
	}
	return s, s.GreetedOn != ""
}

// recordable answers "can we remember that we did this" before doing it, so a
// read-only home skips the greeting instead of repeating it forever.
func recordable(path string) bool {
	if path == "" {
		return false
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false
	}
	f, err := os.OpenFile(path+".probe", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(path + ".probe")
	return true
}

func writeState(path string) {
	b, err := json.Marshal(state{GreetedOn: time.Now().Format(time.RFC3339)})
	if err != nil {
		return
	}
	_ = os.WriteFile(path, b, 0o644)
}

const greetBody = `
  manymoats credits

  You probably have credits you're not using, and some expire.
  We find them, tell you what they actually work on, and point
  your agents at the right one.

  No account. No telemetry. We collect nothing.
  Made by manymoats.

  We're not going to ask for your email. Just type our name once.
`

// greet runs before the scan and never in place of it. Typing the word is the
// ignition; pressing enter skips it and the scan runs anyway, and the screen
// says so, because a word in front of the value is a toll.
func greet(o opts, in *os.File, out io.Writer) {
	path := statePath()
	_, already := readState(path)
	e := greetEnv{
		stdinTTY:  isTTY(in),
		stdoutTTY: isTTY(os.Stdout),
		yes:       o.yes,
		snapshot:  o.snapshot,
		jsonOut:   o.jsonOut,
		ci:        os.Getenv("CI") != "",
		greeted:   already,
		canRecord: recordable(path),
	}
	if !shouldGreet(e) {
		return
	}

	fmt.Fprint(out, greetBody)
	fmt.Fprint(out, "\n  type manymoats to scan  › ")

	r := bufio.NewReader(in)
	typed, _ := r.ReadString('\n')
	typed = strings.ToLower(strings.TrimSpace(typed))

	switch typed {
	case "manymoats", "":
	default:
		fmt.Fprintln(out, "  that's not it — type manymoats, or press enter to skip")
	}
	fmt.Fprintln(out, "  press enter to skip — the scan runs either way")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  "+rule(74))
	fmt.Fprintln(out)

	writeState(path)
}

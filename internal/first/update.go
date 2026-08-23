package first

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Homebrew does not upgrade on its own. These are the honest commands — no
// second tap, no second package manager.
const (
	want    = "want updates while we fix bugs?"
	upgrade = "brew upgrade manymoats"
	tap     = "brew tap homebrew/autoupdate"
	auto    = "brew autoupdate start --upgrade"
)

// Env is the first-run world. Tests pass buffers and fake brew; the binary
// fills it from the real machine.
type Env struct {
	In      io.Reader
	Out     io.Writer
	InTTY   bool
	OutTTY  bool
	CI      bool
	NoColor bool
	NoAnim  bool
	Look    func(string) (string, error)
	Run     func(name string, args ...string) error
}

// Real is the process the user is sitting in.
func Real() Env {
	return Env{
		In:      os.Stdin,
		Out:     os.Stdout,
		InTTY:   isTTY(os.Stdin),
		OutTTY:  isTTY(os.Stdout),
		CI:      os.Getenv("CI") != "",
		NoColor: os.Getenv("NO_COLOR") != "",
		NoAnim:  noAnimFlag(),
		Look:    exec.LookPath,
		Run: func(name string, args ...string) error {
			cmd := exec.Command(name, args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			return cmd.Run()
		},
	}
}

func isTTY(f *os.File) bool {
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

func (e Env) canRecord() bool { return recordable(Path()) }

func (e Env) motion() bool {
	return e.OutTTY && !e.CI && !e.NoColor && !e.NoAnim
}

func (e Env) askable() bool {
	return e.InTTY && e.OutTTY && !e.CI
}

// Hello is the front door: splash, then one question. Later runs skip it.
func Hello(e Env) {
	if Has("main") {
		return
	}
	if e.motion() {
		play(e.Out)
	} else {
		fmt.Fprint(e.Out, Last(true))
	}
	if e.askable() && e.canRecord() {
		if ask(e.In, e.Out) {
			enable(e)
		}
	} else {
		// They cannot press 1 or 2. Show the command; do not wait.
		fmt.Fprintf(e.Out, "\n  %s\n  %s\n\n", want, upgrade)
	}
	if e.canRecord() {
		Mark("main")
	}
}

// App is the smaller hello for orch, credits, eyes, agents. Not the movie.
// Once, and only in a real terminal — a pipe must not grow a banner.
func App(e Env) {
	if Has("main") || Has("apps") {
		return
	}
	if !e.OutTTY || e.CI {
		return
	}
	fmt.Fprint(e.Out, Last(false))
	if e.canRecord() {
		Mark("apps")
	}
}

func ask(in io.Reader, out io.Writer) bool {
	fmt.Fprintf(out, "\n  %s\n\n  1  yes\n  2  no\n\n", want)
	r := bufio.NewReader(in)
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return false
	}
	return strings.TrimSpace(line) == "1"
}

// Update is `manymoats update`. Homebrew's upgrade, or the command to type
// if brew is not on this machine.
func Update(e Env) int {
	if e.Look == nil {
		e.Look = exec.LookPath
	}
	if _, err := e.Look("brew"); err != nil {
		fmt.Fprintln(e.Out, "  "+upgrade)
		return 0
	}
	if e.Run == nil {
		e.Run = Real().Run
	}
	if err := e.Run("brew", "upgrade", "manymoats"); err != nil {
		return 1
	}
	return 0
}

func enable(e Env) {
	if e.Look == nil {
		e.Look = exec.LookPath
	}
	if e.Run == nil {
		e.Run = Real().Run
	}
	if _, err := e.Look("brew"); err != nil {
		fmt.Fprintln(e.Out, "  "+tap)
		fmt.Fprintln(e.Out, "  "+auto)
		return
	}
	if err := e.Run("brew", "autoupdate", "start", "--upgrade"); err == nil {
		return
	}
	if err := e.Run("brew", "tap", "homebrew/autoupdate"); err != nil {
		fmt.Fprintln(e.Out, "  "+tap)
		fmt.Fprintln(e.Out, "  "+auto)
		return
	}
	if err := e.Run("brew", "autoupdate", "start", "--upgrade"); err != nil {
		fmt.Fprintln(e.Out, "  "+tap)
		fmt.Fprintln(e.Out, "  "+auto)
	}
}

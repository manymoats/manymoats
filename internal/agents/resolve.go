package agents

import (
	"fmt"
	"os/exec"
	"strings"
)

// Outcome of trying one pin. "resolved but wrong digest" is deliberately its own
// state: it is the dangerous case, because it succeeds at the thing that looks
// like success.
type Outcome string

const (
	Usable      Outcome = "usable"
	Gone        Outcome = "gone"
	DigestWrong Outcome = "digest differs"
	Untried     Outcome = "not tried"
)

type Try struct {
	Pin     Pin
	Outcome Outcome
	Detail  string
}

// Resolve walks the candidates in order and stops at the first that is usable.
//
// A candidate that resolves with a different digest is REFUSED, not taken. It
// would build something that answers to the name and has never been checked,
// which is a different agent wearing an earned name. The plan calls that out
// and so does the surface.
func Resolve(a Agent, probe func(ref string) (string, error)) (Pin, []Try, bool) {
	tries := make([]Try, 0, len(a.Pins))
	for i, p := range a.Pins {
		got, err := probe(p.Ref)
		switch {
		case err != nil:
			tries = append(tries, Try{p, Gone, shortErr(err)})
		case p.Digest != "" && got != "" && got != p.Digest:
			tries = append(tries, Try{p, DigestWrong, "resolved, but not the file this agent was tested against"})
		default:
			tries = append(tries, Try{p, Usable, ""})
			for _, rest := range a.Pins[i+1:] {
				tries = append(tries, Try{rest, Untried, "an earlier pin was usable"})
			}
			return p, tries, true
		}
	}
	return Pin{}, tries, false
}

func shortErr(err error) string {
	s := err.Error()
	if strings.Contains(s, "404") || strings.Contains(strings.ToLower(s), "not found") {
		return "404"
	}
	if len(s) > 48 {
		s = s[:48]
	}
	return s
}

// OllamaProbe asks the local ollama whether a ref is reachable. It never
// downloads: this is the question "does this still exist", asked before 18 GB
// of someone else's bandwidth is spent on the answer.
func OllamaProbe(ref string) (string, error) {
	out, err := exec.Command("ollama", "show", ref, "--modelfile").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return "", nil
}

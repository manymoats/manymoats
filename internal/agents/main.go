package agents

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

//go:embed data/agents.json
var agentsJSON []byte

func All() ([]Agent, error) { return Load(agentsJSON) }

func find(as []Agent, name string) (Agent, bool) {
	for _, a := range as {
		if a.Name == name {
			return a, true
		}
	}
	return Agent{}, false
}

func installed(name string) bool {
	out, err := exec.Command("ollama", "list").Output()
	return err == nil && strings.Contains(string(out), name)
}

// Main is `manymoats agents`.
func Main() int {
	as, err := All()
	if err != nil {
		fmt.Fprintln(os.Stderr, "manymoats agents:", err)
		return 1
	}
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Print(Roster(as, installed))
		return 0
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Print(help)
		return 0
	case "install":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "manymoats agents: install needs a name")
			return 1
		}
		return install(as, args[1])
	case "verify":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "manymoats agents: verify needs a name")
			return 1
		}
		return verify(as, args[1], runsFlag(args[2:]))
	}
	fmt.Print(Roster(as, installed))
	return 0
}

// runsFlag reads `--runs N`. One run is allowed and is the honest way to get a
// quick look; the surface says how many it asked so a fast check cannot be
// mistaken for a thorough one.
func runsFlag(rest []string) int {
	for i, a := range rest {
		if a == "--runs" && i+1 < len(rest) {
			n, err := strconv.Atoi(rest[i+1])
			if err == nil && n > 0 {
				return n
			}
		}
	}
	return 3
}

func install(as []Agent, name string) int {
	a, ok := find(as, name)
	if !ok {
		fmt.Fprintf(os.Stderr, "manymoats agents: no agent called %q\n", name)
		return 1
	}
	pin, tries, ok := Resolve(a, OllamaProbe)
	if !ok {
		// Refused, never substituted. A candidate that resolves with the wrong
		// digest would build something answering to an earned name.
		fmt.Print(Refused(a, tries))
		return 1
	}
	mf := a.Modelfile(pin)
	fmt.Printf("\n  building %s from %s\n  modelfile %s\n\n", a.Name, pin.Ref, Digest(mf))
	cmd := exec.Command("ollama", "create", a.Name, "-f", "-")
	cmd.Stdin = strings.NewReader(mf)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "  ollama create refused:", err)
		return 1
	}
	fmt.Printf("\n  %s is on your machine. manymoats agents verify %s\n\n", a.Name, a.Name)
	return 0
}

// runs defaults to 3 because the pinned agents run at temperature 0.4. One
// answer is a sample. Three is enough to tell a character flaw from a dice roll,
// and cheap enough that people will actually wait for it.
func verify(as []Agent, name string, runs int) int {
	a, ok := find(as, name)
	if !ok {
		fmt.Fprintf(os.Stderr, "manymoats agents: no agent called %q\n", name)
		return 1
	}
	if !installed(a.Name) {
		fmt.Fprintf(os.Stderr, "\n  %s is not installed, so there is nothing to verify.\n  manymoats agents install %s\n\n", a.Name, a.Name)
		return 1
	}
	results := Verify(a, askOllama(a.Name), runs)
	fmt.Print(Held(a, results))
	for _, r := range results {
		if r.Held < r.Asked {
			return 1
		}
	}
	return 0
}

// askOllama asks the installed model the way the character says it is called.
//
// This went through `ollama run` first, and the system prompt argument was
// accepted and thrown away — so verify was testing whatever happened to be
// baked into the local build rather than the pinned character it reports on.
// The API path sends the pinned system prompt explicitly, and sends
// think:false, which is the condition the character's own sentence names:
// "Thinking is off unless the caller enables it."
//
// `ollama run` enables it. Measured on this Mac, bare CLI stdout opens
// mid-reasoning and closes with a lone </think> — no opening tag, because the
// chat template writes that one before the model generates. That is on the
// known list, not hidden: a caller who does not turn thinking off gets
// reasoning, and no wording in a system prompt can prevent it.
func askOllama(model string) func(string, string) (string, error) {
	return func(system, prompt string) (string, error) {
		msgs := []map[string]string{}
		if system != "" {
			msgs = append(msgs, map[string]string{"role": "system", "content": system})
		}
		msgs = append(msgs, map[string]string{"role": "user", "content": prompt})
		body, err := json.Marshal(map[string]any{
			"model": model, "stream": false, "think": false, "messages": msgs,
			"options": map[string]any{"num_predict": 600},
		})
		if err != nil {
			return "", err
		}
		res, err := http.Post(host()+"/api/chat", "application/json", bytes.NewReader(body))
		if err != nil {
			return "", fmt.Errorf("ollama is not answering on %s: %w", host(), err)
		}
		defer res.Body.Close()
		raw, err := io.ReadAll(res.Body)
		if err != nil {
			return "", err
		}
		if res.StatusCode != 200 {
			return "", fmt.Errorf("ollama said %s: %s", res.Status, trimTo(string(raw), 120))
		}
		var r struct {
			Message struct {
				Content  string `json:"content"`
				Thinking string `json:"thinking"`
			} `json:"message"`
		}
		if err := json.Unmarshal(raw, &r); err != nil {
			return "", err
		}
		// Reasoning that arrives on its own channel is still reasoning the
		// character promised not to produce, so it is folded back in rather
		// than dropped where a claim could not see it.
		return r.Message.Thinking + r.Message.Content, nil
	}
}

func host() string {
	if h := os.Getenv("OLLAMA_HOST"); h != "" {
		if !strings.HasPrefix(h, "http") {
			return "http://" + h
		}
		return h
	}
	return "http://127.0.0.1:11434"
}

func trimTo(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

const help = `manymoats agents — the house's tuned local models, on your machine

  manymoats agents                  which agents exist, which are installed
  manymoats agents install <name>   build it on your ollama, from a checked pin
  manymoats agents verify <name>    ask it to break its own promises
                       --runs N     how many times to ask each way (default 3)

  These run on YOUR machine, on YOUR GPU, at YOUR cost. Nothing here calls
  anything we pay for, and there is no key to ask us for. You bring a machine
  or you bring nothing — that is the whole arrangement.

  An agent whose weights have moved will refuse to build rather than quietly
  substitute a different file. An agent you cannot verify is not the agent.

  The pins run above temperature zero, so one answer is a sample. verify asks
  each way three times and tells you how many held — a claim that wobbles is
  reported as wobbling rather than as a verdict.

`

var _ = json.Marshal

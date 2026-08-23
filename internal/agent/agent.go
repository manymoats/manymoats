package agent

import "time"

type State int

const (
	Asks State = iota
	Working
	Done
	Stalled
	Idle
	Resident
)

func (s State) String() string {
	return [...]string{"asks", "working", "done", "stalled", "idle", "resident"}[s]
}

type Source string

const (
	Claude   Source = "claude"
	Cursor   Source = "cursor"
	Qwen     Source = "qwen"
	Grok     Source = "grok"
	Gemini   Source = "gemini"
	Ollama   Source = "ollama"
	Terminal Source = "terminal"
	Muse     Source = "muse"
)

type Agent struct {
	ID           string
	Source       Source
	Model        string
	Project      string
	State        State
	Since        time.Duration
	TokensMin    float64
	CacheHit     float64
	Sidechain    bool
	Subagents    int
	Effort       string
	Tag          string
	Pot          string
	Free         bool
	VRAMBytes    int64
	CPUPct       float64
	LinesTouched int
}

const UnknownProject = "unknown"

// Paying answers the only money question that matters at a glance: is this agent
// spending a free/included allowance, or is it costing new money right now?
func Paying(src Source, model, entrypoint string) (pot string, free bool) {
	switch src {
	case Ollama, Terminal:
		return "local", true
	case Cursor:
		return "cursor plan", true
	case Claude:
		return "claude plan", true
	case Grok:
		return "grok plan", true
	case Qwen:
		return "studio free", true
	case Gemini:
		return "gcp credit", true
	}
	return "unknown", false
}

// Disambiguate gives a short, stable tag to agents that would otherwise render
// identically. Same model + same project + same state is indistinguishable on a
// board, which makes the board useless for the thing it exists to do.
// RollUpSubagents folds quiet sidechain sessions into their parent as a
// count. A working or asking subagent stays on the board — folding it into
// a parent that is itself idle made live work invisible.
func RollUpSubagents(as []Agent) []Agent {
	kids := map[string]int{}
	for _, a := range as {
		if a.Sidechain {
			kids[a.Project]++
		}
	}
	var out []Agent
	for _, a := range as {
		if a.Sidechain && a.State != Working && a.State != Asks {
			continue
		}
		if !a.Sidechain {
			a.Subagents = kids[a.Project]
		}
		out = append(out, a)
	}
	return out
}

// OnlyActive drops anything that is not doing something right now. A resident
// model sitting in memory is inventory, not activity.
func OnlyActive(as []Agent) []Agent {
	var out []Agent
	for _, a := range as {
		if a.State == Working || a.State == Asks {
			out = append(out, a)
		}
	}
	return out
}

func Disambiguate(as []Agent) {
	seen := map[string]int{}
	for _, a := range as {
		seen[a.Model+"\x00"+a.Project]++
	}
	for i := range as {
		if seen[as[i].Model+"\x00"+as[i].Project] < 2 {
			continue
		}
		id := as[i].ID
		if len(id) >= 4 {
			as[i].Tag = id[:4]
		} else if id != "" {
			as[i].Tag = id
		}
	}
}

// Classify decides what a session is doing. alive is a live process for
// this session. No process → Idle, never Stalled. Stalled is only a
// process that is still running and has gone quiet.
func Classify(lastTurn string, endsQuestion bool, pendingPrompt bool, idle time.Duration, alive bool) State {
	switch {
	case !alive:
		return Idle
	case (endsQuestion || pendingPrompt) && lastTurn == "assistant" && idle > 45*time.Second:
		return Asks
	case idle < 30*time.Second:
		return Working
	case lastTurn == "assistant" && idle <= 15*time.Minute:
		return Done
	case idle > 15*time.Minute:
		return Stalled
	default:
		return Idle
	}
}

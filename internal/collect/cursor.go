package collect

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/manymoats/manymoats/internal/agent"
)

type composer struct {
	ComposerID    string `json:"composerId"`
	LastUpdatedAt int64  `json:"lastUpdatedAt"`
	Unified       string `json:"unifiedMode"`
	Unread        bool   `json:"hasUnreadMessages"`
	Blocking      bool   `json:"hasBlockingPendingActions"`
	PendingPlan   bool   `json:"hasPendingPlan"`
	Archived      bool   `json:"isArchived"`
	Draft         bool   `json:"isDraft"`
	LinesAdded    int    `json:"totalLinesAdded"`
	LinesRemoved  int    `json:"totalLinesRemoved"`
	SubComposers  int    `json:"numSubComposers"`
	Workspace     struct {
		ID  string `json:"id"`
		URI struct {
			FSPath string `json:"fsPath"`
		} `json:"uri"`
	} `json:"workspaceIdentifier"`
}

// Cursor reads Cursor's own chat records rather than guessing from CPU. It gives
// the same measurement Claude gives — when did this last do something — so the
// two are finally comparable, plus a real "waiting on you" signal that CPU could
// never provide.
func Cursor() []agent.Agent {
	sq, err := exec.LookPath("sqlite3")
	if err != nil {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	db := filepath.Join(home, "Library", "Application Support", "Cursor",
		"User", "globalStorage", "state.vscdb")
	if _, err := os.Stat(db); err != nil {
		return nil
	}
	// immutable=1 opens read-only without a lock and without copying 2.5GB
	uri := "file:" + db + "?immutable=1"
	cmd := exec.Command(sq, uri, "SELECT value FROM composerHeaders;")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	// Cursor's own records say WHO and WHAT — project, subagents, blocking prompts,
	// lines touched. They do NOT say "right now": lastUpdatedAt is written at
	// message boundaries, so a chat streaming for half an hour still reports a
	// half-hour-old timestamp. Only CPU knows liveness. Use both, each for the
	// thing it can actually answer.
	cpu := cursorCPU()
	now := time.Now()
	var res []agent.Agent
	for _, ln := range strings.Split(string(out), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(ln), "{") {
			continue
		}
		var c composer
		if json.Unmarshal([]byte(ln), &c) != nil {
			continue
		}
		if c.Archived || c.Draft || c.LastUpdatedAt == 0 {
			continue
		}
		idle := now.Sub(time.UnixMilli(c.LastUpdatedAt))
		if idle > 6*time.Hour {
			continue
		}
		proj := agent.UnknownProject
		if p := c.Workspace.URI.FSPath; p != "" {
			proj = filepath.Base(p)
		}
		st := agent.Classify("assistant", false, c.Blocking || c.PendingPlan || c.Unread, idle, true)
		res = append(res, agent.Agent{
			ID: c.ComposerID, Source: agent.Cursor, Model: "cursor",
			Project: proj, State: st, Since: idle,
			LinesTouched: c.LinesAdded + c.LinesRemoved,
			Subagents:    c.SubComposers,
		})
	}
	// Attach the measurement to the most recently touched chat and let Settle
	// decide the state. A collector that decides for itself has no memory, so a
	// one-second CPU dip drops the row — which is exactly the flicker Settle
	// exists to absorb. Report the number; let the layer with memory judge it.
	if len(res) > 0 {
		best := 0
		for i := range res {
			if res[i].Since < res[best].Since {
				best = i
			}
		}
		res[best].CPUPct = cpu
	}
	return res
}

// cursorCPU sums every Cursor process. The extension host alone reads near-idle
// while the renderers carry a streaming response.
func cursorCPU() float64 {
	out, err := exec.Command("ps", "-eo", "pcpu,args").Output()
	if err != nil {
		return 0
	}
	var total float64
	for _, ln := range strings.Split(string(out), "\n") {
		if !strings.Contains(ln, "Cursor") || strings.Contains(ln, "grep") {
			continue
		}
		f := strings.Fields(ln)
		if len(f) < 1 {
			continue
		}
		if v, err := strconv.ParseFloat(f[0], 64); err == nil {
			total += v
		}
	}
	return total
}

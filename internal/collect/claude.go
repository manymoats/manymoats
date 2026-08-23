package collect

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/manymoats/manymoats/internal/agent"
)

type record struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	CWD       string `json:"cwd"`
	SessionID string `json:"sessionId"`
	Effort    string `json:"effort"`
	Sidechain bool   `json:"isSidechain"`
	Message   struct {
		Model   string `json:"model"`
		Content any    `json:"content"`
		Usage   struct {
			Input      int `json:"input_tokens"`
			Output     int `json:"output_tokens"`
			CacheRead  int `json:"cache_read_input_tokens"`
			CacheWrite int `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

func tailLines(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 8<<20)
	ring := make([]string, 0, n)
	for sc.Scan() {
		if len(ring) == n {
			ring = ring[1:]
		}
		ring = append(ring, sc.Text())
	}
	return ring, sc.Err()
}

func endsWithQuestion(content any) bool {
	var text string
	switch v := content.(type) {
	case string:
		text = v
	case []any:
		for _, blk := range v {
			if m, ok := blk.(map[string]any); ok {
				if t, ok := m["text"].(string); ok {
					text = t
				}
			}
		}
	}
	text = strings.TrimRight(strings.TrimSpace(text), "*_`\"')]}")
	return strings.HasSuffix(text, "?")
}

// ClaudeProjectRoots returns every transcript root we can verify exists.
//
// Official (code.claude.com/docs/en/sessions, /docs/en/claude-directory):
//   - ~/.claude/projects
//   - $CLAUDE_CONFIG_DIR/projects when that env is set
//
// Community-reported second home (better-ccusage; Claude Code issue comments):
//   - ~/.config/claude/projects
//
// Not collected — looked for, not verified as a Claude Design store, or not
// the same JSONL:
//   - ~/.claude.jsonl (does not appear in official docs; official neighbour
//     is ~/.claude.json, which is app state, not a transcript)
//   - ~/Library/Application Support/Claude/claude-code-sessions
//     and local-agent-mode-sessions (Desktop/Cowork metadata or audit.jsonl,
//     different shape)
//   - A distinct "Claude Design" path: none found in public docs.
func ClaudeProjectRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var roots []string
	seen := map[string]bool{}
	add := func(p string) {
		if p == "" {
			return
		}
		p = filepath.Clean(p)
		if seen[p] {
			return
		}
		if st, err := os.Stat(p); err != nil || !st.IsDir() {
			return
		}
		seen[p] = true
		roots = append(roots, p)
	}
	if d := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); d != "" {
		add(filepath.Join(d, "projects"))
	}
	add(filepath.Join(home, ".claude", "projects"))
	add(filepath.Join(home, ".config", "claude", "projects"))
	return roots
}

// ClaudeAll reads every verified projects root. live keys are session IDs
// and/or process working directories; an empty/nil map means no live Claude.
func ClaudeAll(live map[string]bool) ([]agent.Agent, error) {
	var out []agent.Agent
	seen := map[string]bool{}
	for _, root := range ClaudeProjectRoots() {
		as, err := ClaudeSessions(root, live)
		if err != nil {
			return nil, err
		}
		for _, a := range as {
			key := a.ID
			if key == "" {
				key = string(a.Source) + "\x00" + a.Project + "\x00" + a.Model + "\x00" + a.Since.String()
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Since < out[j].Since })
	agent.Disambiguate(out)
	return out, nil
}

// ClaudeSessions reads Claude Code JSONL under root.
//
// Official layout (code.claude.com/docs/en/claude-directory):
//
//	<projects>/<slug>/<session-id>.jsonl
//	<projects>/<slug>/<session-id>/subagents/*.jsonl
//
// The old glob was one directory deep, so a working subagent never reached
// the board. We walk the tree and skip official non-transcript dirs
// (memory/, tool-results/). A file that does not parse as a session is
// dropped, same as before.
//
// live is cwd and/or session ID → true. Nil or empty means no live Claude
// process: those sessions are Idle, never Stalled.
func ClaudeSessions(root string, live map[string]bool) ([]agent.Agent, error) {
	paths, err := claudeJSONL(root)
	if err != nil {
		return nil, err
	}
	var out []agent.Agent
	now := time.Now()
	for _, p := range paths {
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		idle := now.Sub(st.ModTime())
		if idle > 6*time.Hour {
			continue
		}
		lines, err := tailLines(p, 40)
		if err != nil || len(lines) == 0 {
			continue
		}
		var last record
		var model, effort, cwd string
		var sidechain bool
		var cRead, cWrite float64
		var outTok float64
		var firstTS, lastTS time.Time
		for _, l := range lines {
			var r record
			if json.Unmarshal([]byte(l), &r) != nil {
				continue
			}
			if r.Type == "assistant" || r.Type == "user" {
				last = r
			}
			if r.Message.Model != "" {
				model = r.Message.Model
			}
			if r.Effort != "" {
				effort = r.Effort
			}
			if r.CWD != "" {
				cwd = r.CWD
			}
			sidechain = sidechain || r.Sidechain
			if r.Timestamp != "" {
				if ts, e := time.Parse(time.RFC3339, r.Timestamp); e == nil {
					if firstTS.IsZero() || ts.Before(firstTS) {
						firstTS = ts
					}
					if ts.After(lastTS) {
						lastTS = ts
					}
				}
			}
			cRead += float64(r.Message.Usage.CacheRead)
			cWrite += float64(r.Message.Usage.CacheWrite)
			outTok += float64(r.Message.Usage.Output)
		}
		if last.Type == "" {
			continue
		}
		project := agent.UnknownProject
		if cwd != "" {
			project = filepath.Base(cwd)
		}
		cache := 0.0
		if cRead+cWrite > 0 {
			cache = cRead / (cRead + cWrite)
		}
		var tpm float64
		if span := lastTS.Sub(firstTS).Minutes(); span > 0.05 && outTok > 0 {
			tpm = outTok / span
		}
		st8 := agent.Classify(last.Type, endsWithQuestion(last.Message.Content), false, idle, sessionAlive(cwd, last.SessionID, live))
		out = append(out, agent.Agent{
			ID:        last.SessionID,
			Source:    agent.Claude,
			Model:     model,
			Project:   project,
			State:     st8,
			Since:     idle,
			CacheHit:  cache,
			TokensMin: tpm,
			Sidechain: sidechain || isSubagentPath(p),
			Effort:    effort,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Since < out[j].Since })
	agent.Disambiguate(out)
	return out, nil
}

func sessionAlive(cwd, id string, live map[string]bool) bool {
	if len(live) == 0 {
		return false
	}
	if id != "" && live[id] {
		return true
	}
	return cwd != "" && live[cwd]
}

func isSubagentPath(p string) bool {
	return strings.Contains(filepath.ToSlash(p), "/subagents/")
}

func claudeJSONL(root string) ([]string, error) {
	if root == "" {
		return nil, nil
	}
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case "memory", "tool-results":
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	return paths, err
}

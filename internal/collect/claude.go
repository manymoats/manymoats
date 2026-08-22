package collect

import (
	"bufio"
	"encoding/json"
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

func ClaudeSessions(root string, live map[int]bool) ([]agent.Agent, error) {
	pattern := filepath.Join(root, "*", "*.jsonl")
	paths, err := filepath.Glob(pattern)
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
		st8 := agent.Classify(last.Type, endsWithQuestion(last.Message.Content), false, idle, true)
		out = append(out, agent.Agent{
			ID:        last.SessionID,
			Source:    agent.Claude,
			Model:     model,
			Project:   project,
			State:     st8,
			Since:     idle,
			CacheHit:  cache,
			TokensMin: tpm,
			Sidechain: sidechain,
			Effort:    effort,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Since < out[j].Since })
	agent.Disambiguate(out)
	return out, nil
}

package nudge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Note struct {
	TS   string `json:"ts"`
	To   string `json:"to"`
	Text string `json:"text"`
	Read bool   `json:"read"`
}

func path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(home, ".orch")
	if err := os.MkdirAll(d, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(d, "nudges.jsonl"), nil
}

func Send(to, text string) error {
	p, err := path()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	b, _ := json.Marshal(Note{TS: time.Now().UTC().Format(time.RFC3339), To: to, Text: text})
	_, err = f.Write(append(b, '\n'))
	return err
}

// Take returns unread notes for an agent and marks them read. A note is delivered
// exactly once — a nudge that arrives twice reads as the founder repeating himself.
func Take(to string) ([]Note, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var all []Note
	var mine []Note
	for _, ln := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		var n Note
		if json.Unmarshal([]byte(ln), &n) != nil {
			continue
		}
		if !n.Read && (n.To == to || n.To == "" || n.To == "any") {
			mine = append(mine, n)
			n.Read = true
		}
		all = append(all, n)
	}
	if len(mine) == 0 {
		return nil, nil
	}
	var b strings.Builder
	for _, n := range all {
		j, _ := json.Marshal(n)
		b.Write(j)
		b.WriteByte('\n')
	}
	return mine, os.WriteFile(p, []byte(b.String()), 0o600)
}

func Pending(to string) int {
	p, err := path()
	if err != nil {
		return 0
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return 0
	}
	n := 0
	for _, ln := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		var x Note
		if json.Unmarshal([]byte(ln), &x) == nil && !x.Read && (x.To == to || x.To == "" || x.To == "any") {
			n++
		}
	}
	return n
}

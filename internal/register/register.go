package register

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Turn struct {
	TS         string  `json:"ts"`
	SessionID  string  `json:"sessionId"`
	Model      string  `json:"model"`
	Project    string  `json:"project"`
	Effort     string  `json:"effort"`
	Entrypoint string  `json:"entrypoint,omitempty"`
	In         int     `json:"in"`
	Out        int     `json:"out"`
	CacheRead  int     `json:"cache_read"`
	CacheWrite int     `json:"cache_write"`
	Thinking   int     `json:"thinking"`
	Secs       float64 `json:"secs"`
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(home, ".orch")
	if err := os.MkdirAll(d, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(d, "register.jsonl"), nil
}

func Append(t Turn) error {
	if t.SessionID == "" {
		return nil
	}
	if t.TS == "" {
		t.TS = time.Now().UTC().Format(time.RFC3339)
	}
	p, err := Path()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}

package conf

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Machine struct {
	Name string `json:"name"`
	Host string `json:"host"` // "local", or an ssh host alias
}

type Config struct {
	Machines []Machine `json:"machines"`
	Names    string    `json:"names,omitempty"` // model | brand | both
	View     string    `json:"view,omitempty"`  // marks | instrument | waveform | cards
	ShowHost bool      `json:"machinesOn,omitempty"`
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
	return filepath.Join(d, "config.json"), nil
}

func Default() Config {
	return Config{
		Machines: []Machine{{Name: "this mac", Host: "local"}},
		Names:    "model",
		View:     "marks",
	}
}

// Load never fails the app. A missing or broken config yields defaults, because
// a monitor that will not start because of its own settings file is worse than
// one with the wrong nickname on a machine.
func Load() Config {
	p, err := Path()
	if err != nil {
		return Default()
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return Default()
	}
	var c Config
	if json.Unmarshal(b, &c) != nil {
		return Default()
	}
	if len(c.Machines) == 0 {
		c.Machines = Default().Machines
	}
	return c
}

func Save(c Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(b, '\n'), 0o600)
}

// EnsureExists writes a starter config the first time, so the file people are
// meant to edit is already there with their own machines in it.
func EnsureExists(discovered []Machine) Config {
	p, err := Path()
	if err != nil {
		return Default()
	}
	if _, err := os.Stat(p); err == nil {
		return Load()
	}
	c := Default()
	if len(discovered) > 0 {
		c.Machines = discovered
	}
	_ = Save(c)
	return c
}

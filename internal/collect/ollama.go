package collect

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/manymoats/manymoats/internal/agent"
)

type ollamaPS struct {
	Models []struct {
		Name      string `json:"name"`
		SizeVRAM  int64  `json:"size_vram"`
		ExpiresAt string `json:"expires_at"`
	} `json:"models"`
}

func Ollama() []agent.Agent {
	c := &http.Client{Timeout: 900 * time.Millisecond}
	r, err := c.Get("http://localhost:11434/api/ps")
	if err != nil {
		return nil
	}
	defer r.Body.Close()
	var p ollamaPS
	if json.NewDecoder(r.Body).Decode(&p) != nil {
		return nil
	}
	var out []agent.Agent
	for _, m := range p.Models {
		out = append(out, agent.Agent{
			ID: m.Name, Source: agent.Ollama, Model: m.Name, Project: "local",
			State: agent.Resident, VRAMBytes: m.SizeVRAM,
		})
	}
	return out
}

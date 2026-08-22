package orch

import (
	"fmt"
	"testing"
	"time"

	"github.com/manymoats/manymoats/internal/agent"
)

func TestDemoMultiProject(t *testing.T) {
	as := []agent.Agent{
		{Source: agent.Claude, Model: "claude-opus-5", Project: "manyAPPS-kpf", State: agent.Working, TokensMin: 11600, Since: time.Second},
		{Source: agent.Cursor, Model: "cursor", Project: "manymoats-kpf", State: agent.Working, LinesTouched: 18341, Since: 30 * time.Second, Subagents: 3},
		{Source: agent.Cursor, Model: "cursor", Project: "manymoats-kpf", State: agent.Asks, Since: 2 * time.Minute},
		{Source: agent.Qwen, Model: "qwen3.8-max", Project: "lathe", State: agent.Working, TokensMin: 4200, Since: 5 * time.Second},
		{Source: agent.Grok, Model: "grok", Project: "filefriend", State: agent.Working, TokensMin: 800, Since: time.Minute},
	}
	fmt.Print(model{w: 100, h: 40, agents: as, view: viewMarks}.View())
}

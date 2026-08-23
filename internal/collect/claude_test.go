package collect

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/manymoats/manymoats/internal/agent"
)

func writeJSONL(t *testing.T, path, body string, age time.Duration) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	at := time.Now().Add(-age)
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
}

const assistantLine = `{"type":"assistant","timestamp":"2026-08-23T01:00:00Z","cwd":"/tmp/kpf","sessionId":"11111111-1111-1111-1111-111111111111","message":{"model":"claude-opus-5","content":"done"}}`

func TestOldJSONLWithoutProcessIsIdleNotStalled(t *testing.T) {
	root := t.TempDir()
	writeJSONL(t, filepath.Join(root, "-tmp-kpf", "s1.jsonl"), assistantLine, time.Hour)
	as, err := ClaudeSessions(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(as) != 1 {
		t.Fatalf("got %d sessions", len(as))
	}
	if as[0].State != agent.Idle {
		t.Fatalf("old jsonl + no process must be idle, got %s", as[0].State)
	}
}

func TestOldJSONLWithLiveCWDCanStall(t *testing.T) {
	root := t.TempDir()
	writeJSONL(t, filepath.Join(root, "-tmp-kpf", "s1.jsonl"), assistantLine, time.Hour)
	as, err := ClaudeSessions(root, map[string]bool{"/tmp/kpf": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(as) != 1 || as[0].State != agent.Stalled {
		t.Fatalf("a live process that has gone quiet is stalled, got %+v", as)
	}
}

func TestNestedSubagentJSONLIsCollected(t *testing.T) {
	root := t.TempDir()
	line := `{"type":"assistant","timestamp":"2026-08-23T01:00:00Z","cwd":"/tmp/kpf","sessionId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","isSidechain":true,"message":{"model":"claude-opus-5","content":"working"}}`
	writeJSONL(t, filepath.Join(root, "-tmp-kpf", "parent", "subagents", "agent-1.jsonl"), line, 5*time.Second)
	as, err := ClaudeSessions(root, map[string]bool{"/tmp/kpf": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(as) != 1 {
		t.Fatalf("nested subagent jsonl must be collected, got %d", len(as))
	}
	if as[0].State != agent.Working {
		t.Fatalf("fresh live subagent must be working, got %s", as[0].State)
	}
	if !as[0].Sidechain {
		t.Fatal("subagents/ path is a sidechain")
	}
}

func TestMemoryAndToolResultsAreNotSessions(t *testing.T) {
	root := t.TempDir()
	writeJSONL(t, filepath.Join(root, "-tmp-kpf", "memory", "note.jsonl"), assistantLine, time.Minute)
	writeJSONL(t, filepath.Join(root, "-tmp-kpf", "sid", "tool-results", "big.jsonl"), assistantLine, time.Minute)
	as, err := ClaudeSessions(root, map[string]bool{"/tmp/kpf": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(as) != 0 {
		t.Fatalf("memory/ and tool-results/ are not sessions, got %d", len(as))
	}
}

func TestIsClaudeCLI(t *testing.T) {
	if !isClaudeCLI("/usr/local/bin/claude --resume abc") {
		t.Fatal("the claude binary is a live Claude")
	}
	if !isClaudeCLI("node /opt/@anthropic-ai/claude-code/cli.js") {
		t.Fatal("the published cli.js is a live Claude")
	}
	if isClaudeCLI("/Applications/Claude.app/Contents/MacOS/Claude") {
		t.Fatal("the Desktop app is not one session")
	}
	if isClaudeCLI("chrome https://claude.ai") {
		t.Fatal("a browser tab is not a Claude process")
	}
}

func TestResumeSessionID(t *testing.T) {
	id := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	if got := resumeSessionID("claude --resume " + id); got != id {
		t.Fatalf("got %q", got)
	}
	if got := resumeSessionID("claude -p hi"); got != "" {
		t.Fatalf("no id, got %q", got)
	}
}

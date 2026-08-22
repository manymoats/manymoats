package agent

import "testing"

func TestOneLaneNotTwo(t *testing.T) {
	as := []Agent{
		{Source: Claude, Model: "claude-opus-5"},
		{Source: Cursor, Model: "cursor-composer-2"},
		{Source: Qwen, Model: "qwen/qwen3.8-max"},
		{Source: Ollama, Model: "big:latest"},
	}
	for _, a := range as {
		if got := Display(a, NameModel); got == "" || got == string(a.Source) {
			t.Errorf("%s: model lane returned %q — every row must name a MODEL, never fall back to the brand", a.Source, got)
		}
		if got := Display(a, NameBrand); got != Brand(a.Source) {
			t.Errorf("%s: brand lane returned %q", a.Source, got)
		}
	}
}

func TestMarkNeverRepeatsItselfInText(t *testing.T) {
	if got := ModelName("claude-opus-5", Claude); got != "opus-5" {
		t.Fatalf("the mark already says claude; text got %q, want opus-5", got)
	}
	if got := ModelName("qwen/qwen3.8-max", Qwen); got != "qwen3.8-max" {
		t.Fatalf("got %q", got)
	}
}

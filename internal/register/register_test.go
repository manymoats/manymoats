package register

import (
	"encoding/json"
	"testing"
)

func TestSessionIDIsRecorded(t *testing.T) {
	b, _ := json.Marshal(Turn{SessionID: "abc", Model: "opus", Project: "p"})
	var m map[string]any
	json.Unmarshal(b, &m)
	if m["sessionId"] != "abc" {
		t.Fatal("sessionId must be on every row — two agents in one project are otherwise inseparable (panel R4)")
	}
}

func TestRowsWithoutASessionAreDropped(t *testing.T) {
	if err := Append(Turn{Model: "opus"}); err != nil {
		t.Fatalf("a session-less row should be silently skipped, not error: %v", err)
	}
}

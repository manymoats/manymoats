package first

import (
	"os"
	"path/filepath"
	"testing"
)

func isolate(t *testing.T) {
	t.Helper()
	d := t.TempDir()
	t.Setenv("HOME", d)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(d, ".config"))
}

func TestAFreshMachineHasNotBeenSeen(t *testing.T) {
	isolate(t)
	if Has("main") || Has("apps") {
		t.Fatal("a new home must not look like a first run already happened")
	}
}

func TestTheChoiceIsRemembered(t *testing.T) {
	isolate(t)
	Mark("main")
	if !Has("main") {
		t.Fatal("after the first run, the file must say so")
	}
	p := Path()
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("the seen file should exist at %s: %v", p, err)
	}
}

func TestDeletingTheFileBringsThePromptBack(t *testing.T) {
	isolate(t)
	Mark("main")
	if err := os.Remove(Path()); err != nil {
		t.Fatal(err)
	}
	if Has("main") {
		t.Fatal("deleting the seen file is how you see the first run again")
	}
}

func TestAppsNoteDoesNotConsumeTheFrontDoor(t *testing.T) {
	isolate(t)
	Mark("apps")
	if !Has("apps") {
		t.Fatal("the smaller hello should be remembered")
	}
	if Has("main") {
		t.Fatal("opening an app first must not spend the main splash")
	}
}

func TestAnUnwritableHomeIsNotRecordable(t *testing.T) {
	if recordable("") {
		t.Fatal("nowhere to write is not recordable")
	}
}

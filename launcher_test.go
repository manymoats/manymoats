package main

import (
	"strings"
	"testing"

	"github.com/manymoats/manymoats/internal/launcher"
)

// The founder types "manymoats -- orch". It printed nothing at all, because "--"
// was read as the name of an app. "--" is also the POSIX end-of-options marker,
// so refusing it was wrong twice over.
func TestSeparatorAndDashedFormsReachTheApp(t *testing.T) {
	all := apps()
	for _, in := range [][]string{
		{"orch"},
		{"--", "orch"},
		{"--orch"},
		{"--", "credits", "--snapshot"},
		{"--credits"},
	} {
		got := normalise(in, all)
		if len(got) == 0 {
			t.Fatalf("%v resolved to nothing", in)
		}
		found := false
		for _, a := range all {
			if a.Name == got[0] {
				found = true
			}
		}
		if !found {
			t.Errorf("%v → %q, which is not an app", in, got[0])
		}
	}
	// a bare "--" with nothing after it is the directory, not an error
	if len(normalise([]string{"--"}, all)) != 0 {
		t.Error(`"--" alone should fall through to the directory`)
	}
}

func TestNoAnimIsNotAnApp(t *testing.T) {
	got := dropNoAnim(normalise([]string{"--no-anim"}, apps()))
	if len(got) != 0 {
		t.Fatalf("--no-anim should fall through to the directory, got %v", got)
	}
}

func TestTheDirectoryNameIsLowercase(t *testing.T) {
	got := launcher.Directory(apps())
	if strings.Contains(got, "MANYMOATS") || strings.Contains(got, "ManyMoats") || strings.Contains(got, "ManyMotes") {
		t.Fatal("the word is manymoats, never a capital M")
	}
	if !strings.Contains(got, "manymoats") {
		t.Fatal("the directory must say manymoats")
	}
	if !strings.Contains(got, "by manymoats") {
		t.Fatal("the directory must finish with the credit")
	}
}

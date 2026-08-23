package first

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func strip(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' && s[i] != 'J' && s[i] != 'H' && s[i] != 'l' && s[i] != 'h' {
				i++
			}
			if i < len(s) {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func still(in io.Reader) Env {
	if in == nil {
		in = strings.NewReader("")
	}
	return Env{
		In:     in,
		Out:    &bytes.Buffer{},
		NoAnim: true,
		Look:   func(string) (string, error) { return "", errors.New("no brew") },
		Run:    func(string, ...string) error { return errors.New("no brew") },
	}
}

func TestHelloSkipsOnceSeen(t *testing.T) {
	isolate(t)
	Mark("main")
	e := still(strings.NewReader("1\n"))
	e.InTTY, e.OutTTY = true, true
	Hello(e)
	got := e.Out.(*bytes.Buffer).String()
	if got != "" {
		t.Fatalf("a second run must stay quiet, got %q", got)
	}
}

func TestHelloDoesNotBlockAPipe(t *testing.T) {
	isolate(t)
	e := still(nil)
	Hello(e)
	got := strip(e.Out.(*bytes.Buffer).String())
	if !strings.Contains(got, hero) {
		t.Fatal("a pipe still gets the last frame — the name must appear")
	}
	if !strings.Contains(got, credit) {
		t.Fatal("a pipe still gets the credit")
	}
	if !strings.Contains(got, upgrade) {
		t.Fatal("when they cannot press 1 or 2, show the exact command")
	}
	if !Has("main") {
		t.Fatal("a pipe that showed the command must be remembered, or CI prints it forever")
	}
}

func TestCIDoesNotHang(t *testing.T) {
	isolate(t)
	e := still(strings.NewReader("this would block if anyone read a lot of it\n"))
	e.CI = true
	e.InTTY, e.OutTTY = true, true
	Hello(e)
	got := strip(e.Out.(*bytes.Buffer).String())
	if strings.Contains(got, "1  yes") {
		t.Fatal("CI must not wait for a key")
	}
	if !strings.Contains(got, upgrade) {
		t.Fatal("CI should still see the command")
	}
}

func TestYesWithoutBrewPrintsTheTwoCommands(t *testing.T) {
	isolate(t)
	e := still(strings.NewReader("1\n"))
	e.InTTY, e.OutTTY = true, true
	Hello(e)
	got := strip(e.Out.(*bytes.Buffer).String())
	if !strings.Contains(got, want) {
		t.Fatal("the question is missing")
	}
	if !strings.Contains(got, tap) || !strings.Contains(got, auto) {
		t.Fatalf("yes without brew must print the two commands, got:\n%s", got)
	}
	if !Has("main") {
		t.Fatal("answering must be remembered")
	}
}

func TestNoIsRememberedAndDoesNotEnable(t *testing.T) {
	isolate(t)
	var ran []string
	e := still(strings.NewReader("2\n"))
	e.InTTY, e.OutTTY = true, true
	e.Look = func(string) (string, error) { return "/opt/homebrew/bin/brew", nil }
	e.Run = func(name string, args ...string) error {
		ran = append(ran, name+" "+strings.Join(args, " "))
		return nil
	}
	Hello(e)
	if len(ran) != 0 {
		t.Fatalf("no must not run brew, ran %v", ran)
	}
	if !Has("main") {
		t.Fatal("no must still be remembered, or the question comes back every time")
	}
}

func TestUpdatePrintsTheCommandWhenBrewIsMissing(t *testing.T) {
	var buf bytes.Buffer
	e := still(nil)
	e.Out = &buf
	if code := Update(e); code != 0 {
		t.Fatalf("missing brew is not a failure, exit %d", code)
	}
	if !strings.Contains(buf.String(), upgrade) {
		t.Fatalf("wanted %q, got %q", upgrade, buf.String())
	}
}

func TestYesWithBrewStartsAutoupdate(t *testing.T) {
	isolate(t)
	var ran []string
	e := still(strings.NewReader("1\n"))
	e.InTTY, e.OutTTY = true, true
	e.Look = func(string) (string, error) { return "/opt/homebrew/bin/brew", nil }
	e.Run = func(name string, args ...string) error {
		ran = append(ran, name+" "+strings.Join(args, " "))
		return nil
	}
	Hello(e)
	if len(ran) != 1 || ran[0] != "brew autoupdate start --upgrade" {
		t.Fatalf("yes should start autoupdate, ran %v", ran)
	}
}

func TestEnableTapsWhenAutoupdateIsMissing(t *testing.T) {
	isolate(t)
	var ran []string
	e := still(strings.NewReader("1\n"))
	e.InTTY, e.OutTTY = true, true
	e.Look = func(string) (string, error) { return "/opt/homebrew/bin/brew", nil }
	e.Run = func(name string, args ...string) error {
		cmd := name + " " + strings.Join(args, " ")
		ran = append(ran, cmd)
		if cmd == "brew autoupdate start --upgrade" && len(ran) == 1 {
			return errors.New("unknown command")
		}
		return nil
	}
	Hello(e)
	if len(ran) < 2 || ran[1] != "brew tap homebrew/autoupdate" {
		t.Fatalf("missing autoupdate should tap homebrew/autoupdate, ran %v", ran)
	}
}

func TestUpdateRunsBrewUpgrade(t *testing.T) {
	var got []string
	e := still(nil)
	e.Look = func(string) (string, error) { return "/opt/homebrew/bin/brew", nil }
	e.Run = func(name string, args ...string) error {
		got = append(got, append([]string{name}, args...)...)
		return nil
	}
	if code := Update(e); code != 0 {
		t.Fatalf("exit %d", code)
	}
	want := []string{"brew", "upgrade", "manymoats"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("ran %v, want %v", got, want)
	}
}

func TestAppIsSmallAndDoesNotReplayTheMovie(t *testing.T) {
	isolate(t)
	e := still(nil)
	e.OutTTY = true
	App(e)
	got := strip(e.Out.(*bytes.Buffer).String())
	if !strings.Contains(got, credit) {
		t.Fatal("the smaller hello must finish with the credit")
	}
	if strings.Contains(got, "▟█████▙") {
		t.Fatal("a sub-app must not play the tall movie")
	}
	if Has("main") {
		t.Fatal("a sub-app must not spend the main splash")
	}
	if !Has("apps") {
		t.Fatal("the smaller hello should be remembered")
	}

	e2 := still(nil)
	e2.OutTTY = true
	App(e2)
	if e2.Out.(*bytes.Buffer).Len() != 0 {
		t.Fatal("the smaller hello is once")
	}

	e3 := still(strings.NewReader("2\n"))
	e3.InTTY, e3.OutTTY = true, true
	Hello(e3)
	if !strings.Contains(strip(e3.Out.(*bytes.Buffer).String()), hero) {
		t.Fatal("the front door still plays after an app was opened first")
	}
}

func TestAppStaysQuietOnAPipe(t *testing.T) {
	isolate(t)
	e := still(nil)
	App(e)
	if e.Out.(*bytes.Buffer).Len() != 0 {
		t.Fatal("a pipe must not grow a banner")
	}
}

func TestSplashLastFrameIsTheNameAndCredit(t *testing.T) {
	got := strip(Last(true))
	if !strings.Contains(got, hero) {
		t.Fatal("the last frame must read manymoats")
	}
	if !strings.Contains(got, credit) {
		t.Fatal("the last frame must finish with by manymoats")
	}
	if strings.Contains(got, "ORCH") || strings.Contains(got, "O R C H") {
		t.Fatal("the front door hero is manymoats, not orch")
	}
}

func TestSplashStartsWithoutTheWord(t *testing.T) {
	got := strip(mainFrame(0, longMS, 0))
	if strings.Contains(got, hero) {
		t.Fatal("at the first beat the word has not arrived yet — that is the animation")
	}
}

func TestBrandIsLowercaseEverywhereWePrint(t *testing.T) {
	if hero != "manymoats" {
		t.Fatalf("hero is %q", hero)
	}
	if credit != "by manymoats" {
		t.Fatalf("credit is %q", credit)
	}
	for _, s := range []string{hero, credit, want, upgrade, tap, auto, Last(true), Last(false)} {
		for _, banned := range []string{"MANYMOATS", "ManyMoats", "ManyMotes", "Many Brew", "ManyBrew"} {
			if strings.Contains(s, banned) {
				t.Errorf("banned brand form %q", banned)
			}
		}
	}
}

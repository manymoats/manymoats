package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/manymoats/manymoats/internal/fc"
	"github.com/manymoats/manymoats/internal/launcher"
	"github.com/manymoats/manymoats/internal/orch"
	"github.com/manymoats/manymoats/internal/version"
	"github.com/muesli/termenv"
)

func apps() []launcher.App {
	return []launcher.App{
		{Name: "orch", What: "who's working, which model, what it costs", Ready: true, Run: orch.Main},
		{Name: "credits", What: "what's actually free, and what only looks free", Ready: true, Run: fc.Main},
	}
}

// normalise accepts the forms a person actually types. "manymoats -- orch" is
// the founder's own phrasing and "--" is the POSIX end-of-options marker, but it
// was being read as the name of an app, so the command printed nothing at all.
func normalise(args []string, all []launcher.App) []string {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) > 0 {
		for _, a := range all {
			if args[0] == "--"+a.Name {
				args[0] = a.Name
			}
		}
	}
	return args
}

func main() {
	if os.Getenv("NO_COLOR") == "" {
		lipgloss.SetColorProfile(termenv.TrueColor)
	}
	all := apps()
	args := os.Args[1:]

	args = normalise(args, all)

	if len(args) == 0 {
		fmt.Print(launcher.Directory(all))
		return
	}

	switch args[0] {
	case "-v", "--version", "version":
		fmt.Println("manymoats " + version.V)
		return
	case "-h", "--help", "help":
		fmt.Print(launcher.Directory(all))
		return
	}

	for _, a := range all {
		if a.Name != args[0] {
			continue
		}
		if !a.Ready || a.Run == nil {
			fmt.Printf("\n  %s is not built yet.\n\n", a.Name)
			os.Exit(1)
		}
		// Drop the subcommand so the app sees its own flags at the position it
		// expects, exactly as if it had been invoked directly.
		os.Args = append([]string{os.Args[0] + " " + a.Name}, args[1:]...)
		os.Exit(a.Run())
	}

	fmt.Print(launcher.Unknown(args[0], all))
	os.Exit(1)
}

package eyes

import (
	"fmt"
	"os"
	"strconv"
)

// Main is `manymoats eyes`. Returns an exit code; the dispatcher owns teardown.
func Main() int {
	args := os.Args[1:]
	frames, limit := 1, 80
	var subject []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--frames":
			if i+1 < len(args) {
				frames, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "--width":
			if i+1 < len(args) {
				limit, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "-h", "--help":
			fmt.Print(help)
			return 0
		default:
			subject = append(subject, args[i])
		}
	}
	if len(subject) == 0 {
		fmt.Print(empty)
		return 0
	}
	r, err := Look(subject, frames, limit)
	if err != nil {
		fmt.Fprintln(os.Stderr, "manymoats eyes:", err)
		return 1
	}
	fmt.Print(r.Render())
	for _, c := range r.Claims {
		if c.Verdict == Disagrees {
			return 1
		}
	}
	return 0
}

const help = `manymoats eyes — does this screen say what it measures?

  manymoats eyes <command...>        look at what that command prints
    --frames N                       capture N frames and compare them
    --width N                        the column limit to check against (default 80)

  Freeze the subject before asking about motion. A command that re-reads live
  data prints different digits every run, and that difference is not animation.

`

const empty = `
  EYES

  nothing measured yet

  point it at something:  manymoats eyes <command...>

  An empty report is not a clean bill. It means no claim was checked,
  which is a different thing from every claim agreeing.

  by manymoats

`

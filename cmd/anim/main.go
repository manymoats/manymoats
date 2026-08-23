package main

import (
	"fmt"
	"os"

	"github.com/manymoats/manymoats/internal/orch"
)

func main() {
	only := -1
	if len(os.Args) > 1 {
		fmt.Sscanf(os.Args[1], "%d", &only)
	}
	for i := 0; i < 32; i++ {
		if only >= 0 && i != only {
			continue
		}
		fmt.Printf("\n── frame %02d ─────────────────────────────────────\n%s", i+1, orch.RenderCut(i, 96, 48))
	}
}

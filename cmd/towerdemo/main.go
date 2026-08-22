package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/manymoats/manymoats/internal/agent"
)

func paint(hex, s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render(s)
}

func main() {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#2a3038"))
	ground := agent.DotGround(54, 9, 4)

	fmt.Println()
	for i, row := range agent.Tower {
		g := ground[i]
		line := "   " + agent.Metal(row, 0, paint)
		pad := 40 - len([]rune(row)) - 3
		if pad < 0 {
			pad = 0
		}
		line += strings.Repeat(" ", 4)
		if i == 2 {
			line += agent.Metal("O R C H", 0, paint)
		}
		if i == 3 {
			line += dim.Render("the lookout")
		}
		if i == 5 {
			line += dim.Render("by manymoats")
		}
		_ = g
		_ = pad
		fmt.Println(line)
	}
	fmt.Println()
	fmt.Println("  the silver ramp, swept:")
	for p := 0; p < 4; p++ {
		fmt.Println("   " + agent.Metal("████████████████████████", p*2, paint))
	}
	fmt.Println()
	fmt.Println("  dot ground at three spacings:")
	for _, sp := range []int{3, 4, 6} {
		rows := agent.DotGround(48, 3, sp)
		for _, r := range rows {
			fmt.Println("   " + dim.Render(r) + "  ")
		}
		fmt.Printf("   %s\n\n", dim.Render(fmt.Sprintf("spacing %d", sp)))
	}
}

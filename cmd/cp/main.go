package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func main() {
	fmt.Println("detected profile:", lipgloss.ColorProfile())
	fmt.Println("hasDarkBackground:", lipgloss.HasDarkBackground())
}

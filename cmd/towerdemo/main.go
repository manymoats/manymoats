package main

import (
	"fmt"

	"github.com/manymoats/manymoats/internal/first"
	"github.com/manymoats/manymoats/internal/orch"
)

func main() {
	fmt.Println("house last still:")
	fmt.Print(first.Last(true))
	fmt.Println("orch last still:")
	fmt.Print(orch.LastStill(96, 48))
}

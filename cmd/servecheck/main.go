package main

import (
	"fmt"

	"github.com/manymoats/manymoats/internal/serve"
)

func main() {
	bind, iface, err := serve.Resolve(2222)
	if err != nil {
		fmt.Printf("  orch serve would REFUSE TO START: %v\n", err)
		return
	}
	fmt.Printf("  would bind %s on %s\n", bind, iface)
}

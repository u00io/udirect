package main

import (
	"fmt"
	"os"

	example02u00test "github.com/u00io/udirect/examples/example_02_u00_test"
)

func main() {
	// args: --server or --client:x.x.x.x
	if len(os.Args) > 1 && os.Args[1] == "--server" {
		example02u00test.Run(true, "")
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "--client" {
		if len(os.Args) < 3 {
			fmt.Println("Usage: udirect --client x.x.x.x")
			return
		}

		example02u00test.Run(false, os.Args[2])
		return
	}
	fmt.Println("Usage: udirect --server|--client x.x.x.x")
}

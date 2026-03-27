package main

import (
	"fmt"
	"os"

	example03uapi "github.com/u00io/udirect/examples/example_03_uapi"
)

func main() {
	// args: --server or --client:x.x.x.x
	if len(os.Args) > 1 && os.Args[1] == "--server" {
		example03uapi.Run(true, "")
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "--client" {
		if len(os.Args) < 3 {
			fmt.Println("Usage: udirect --client x.x.x.x")
			return
		}

		example03uapi.Run(false, os.Args[2])
		return
	}
	fmt.Println("Usage: udirect --server|--client x.x.x.x")
}

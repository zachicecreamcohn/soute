package main

import (
	"fmt"
	"os"

	"soute/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "soute:", err)
		os.Exit(1)
	}
}

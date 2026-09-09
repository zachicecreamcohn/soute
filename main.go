package main

import (
	"fmt"
	"os"

	"github.com/zachicecreamcohn/soute/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "soute:", err)
		os.Exit(1)
	}
}

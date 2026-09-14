// Command soute is a zero-friction project file versioning CLI.
package main

import (
	"fmt"
	"os"

	"github.com/zachicecreamcohn/soute/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "soute:", err)
		os.Exit(1)
	}
}

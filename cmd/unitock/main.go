// Command unitock is a single-task terminal time tracker.
package main

import (
	"fmt"
)

// Set at build time with -ldflags "-X main.version=...".
var version, commit, date = "dev", "none", "unknown"

func main() {
	fmt.Printf("unitock %s (%s, %s)\n", version, commit, date)
}

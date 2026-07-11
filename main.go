package main

import (
	"fmt"
	"os"
)

// version is stamped by goreleaser via -ldflags at release time.
var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "usher:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 && (args[0] == "version" || args[0] == "--version") {
		fmt.Println("usher", version)
		return nil
	}
	fmt.Println("usher — right this way. (v0.1 under construction)")
	return nil
}

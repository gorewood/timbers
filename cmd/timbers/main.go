// Package main provides the entry point for the timbers CLI.
package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("timbers", version)
	fmt.Println("A Git-native development ledger")
	return nil
}

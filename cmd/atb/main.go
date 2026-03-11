// Package main provides the atb command entrypoint.
package main

import "os"

var version = "dev"

func main() {
	if err := Execute(); err != nil {
		os.Exit(1)
	}
}

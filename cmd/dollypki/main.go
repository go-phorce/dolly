// Command dollypki is a command-line utility for managing HSM keys and creating certificates
package main

import (
	"os"

	"github.com/go-phorce/dolly/cmd/dollypki/pkg"
)

func main() {
	// Logs are set to os.Stderr, while output to os.Stdout
	rc := pkg.ParseAndRun("dollypki", os.Args, os.Stdout)
	os.Exit(int(rc))
}

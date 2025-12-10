package main

import (
	"os"
	"testing"
)

// TestMainEntry exercises the top-level main() function. We don't assert on
// output here; the goal is to ensure the CLI entrypoint can be invoked without
// panicking or exiting in the default case (no subcommands provided).
func TestMainEntry(t *testing.T) {
	// Save and restore os.Args to avoid polluting global state across tests.
	orig := os.Args
	t.Cleanup(func() { os.Args = orig })
	os.Args = []string{"todototum"}

	// Expect no panic or abnormal termination when invoking main.
	main()
}

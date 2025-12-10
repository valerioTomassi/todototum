package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"testing"

	"github.com/spf13/cobra"
)

// TestExecute_Success exercises the happy path where the root command runs
// without errors. Cobra will print usage and return nil when no subcommand is
// provided, so we simply ensure the function does not call os.Exit.
func TestExecute_Success(t *testing.T) {
	// Save and restore original args to avoid leaking state between tests.
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	// Keep argv[0] intact to avoid breaking subprocess discovery in other tests.
	os.Args = []string{origArgs[0]}
	// Tell cobra to treat as no-args invocation.
	rootCmd.SetArgs(nil)

	// This should not exit the process and should not return an error internally.
	Execute()
}

// TestExecute_ErrorExit verifies the error branch which triggers os.Exit(1).
// We run the test in a subprocess so that exiting the process doesn't kill the
// parent test runner. Inside the child process we add a transient subcommand
// that fails and then invoke Execute().
func TestExecute_ErrorExit(t *testing.T) {
	if os.Getenv("WANT_EXECUTE_ERROR") == "1" {
		// Build a subcommand that returns an error when executed.
		boom := &cobra.Command{
			Use:  "boom",
			RunE: func(cmd *cobra.Command, args []string) error { return assertErr("kaboom") },
		}
		rootCmd.AddCommand(boom)
		defer func() { // cleanup so this subcommand doesn't leak to other tests
			for i, c := range rootCmd.Commands() {
				if c.Use == "boom" {
					rootCmd.RemoveCommand(rootCmd.Commands()[i])
					break
				}
			}
		}()

		// Force cobra to run the failing subcommand using SetArgs to avoid messing with os.Args.
		rootCmd.SetArgs([]string{"boom"})
		Execute() // expect process to exit with status 1
		return
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable failed: %v", err)
	}
	cmd := exec.Command(exe, "-test.run", "TestExecute_ErrorExit")
	cmd.Env = append(os.Environ(), "WANT_EXECUTE_ERROR=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err == nil {
		t.Fatalf("expected non-nil error (process should exit), stderr: %s", stderr.String())
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.Success() {
		t.Fatalf("expected failing exit status, got: %v, stderr: %s", err, stderr.String())
	}
}

// assertErr is a tiny helper to create an error inline without importing fmt
// in the child process block where imports are fixed.
func assertErr(msg string) error { return &simpleErr{s: msg} }

type simpleErr struct{ s string }

func (e *simpleErr) Error() string { return e.s }

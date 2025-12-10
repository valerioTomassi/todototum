package todo

import (
	"fmt"
	"testing"
)

// badCloser simulates a Close() failure
type badCloser struct{}

func (b badCloser) Close() error { return fmt.Errorf("mock close fail") }

func TestSafeClose_LogsError(t *testing.T) {
	c := badCloser{}
	// This should trigger the SafeClose warning branch.
	// No panic or crash = success.
	SafeClose(c, "dummy.txt")
}

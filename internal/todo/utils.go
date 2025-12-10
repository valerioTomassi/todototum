package todo

import (
	"fmt"
	"io"
	"os"
)

// SafeClose closes the provided io.Closer and logs a warning to stderr if it
// fails. It's safe to use in deferred statements for readers and writers.
func SafeClose(c io.Closer, context string) {
	if err := c.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: closing %s: %v\n", context, err)
	}
}

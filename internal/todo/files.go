package todo

import (
	"io"
	"os"
)

// FileReader abstracts opening files to allow testing over in-memory or
// alternate backends.
type FileReader interface {
	Open(name string) (io.ReadCloser, error)
}

// OSFileReader implements FileReader using the real os package.
type OSFileReader struct{}

// Open opens a file from disk.
func (OSFileReader) Open(name string) (io.ReadCloser, error) {
	return os.Open(name)
}

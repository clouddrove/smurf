package terraform

import (
	"bytes"
	"io"
)

// CustomColorWriter is a custom io.Writer that colors lines based on their prefix
// + for additions, - for deletions, ~ for changes, and no prefix for unchanged lines
// It also skips empty lines
type CustomColorWriter struct {
	Buffer *bytes.Buffer
	Writer io.Writer
}

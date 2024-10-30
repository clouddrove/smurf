package configs

import (
	"bytes"
	"io"
)

type CustomColorWriter struct {
	Buffer *bytes.Buffer
	Writer io.Writer
}

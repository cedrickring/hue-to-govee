package logger

import (
	"io"
	"log"
)

// Discard returns a logger that discards all output.
func Discard() *log.Logger {
	return log.New(io.Discard, "", 0)
}

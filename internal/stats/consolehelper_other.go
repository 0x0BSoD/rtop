//go:build !windows

package stats

import (
	"io"
	"os"
)

func clearConsole() {}

func getOutput() io.Writer {
	return os.Stdout
}

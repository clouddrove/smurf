package configs

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/fatih/color"
)

func (w *CustomColorWriter) Write(p []byte) (n int, err error) {
	w.Buffer.Write(p)

	scanner := bufio.NewScanner(bytes.NewReader(p))

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if len(strings.TrimSpace(line)) == 0 {
			fmt.Fprintln(w.Writer)
			continue
		}

		// Color the line based on its prefix
		var coloredLine string

		// Trim any leading whitespace to check the first actual character
		trimmedLine := strings.TrimLeft(line, " ")
		if len(trimmedLine) > 0 {
			switch trimmedLine[0] {
			case '+':
				coloredLine = color.GreenString(line)
			case '-':
				coloredLine = color.RedString(line)
			case '~':
				coloredLine = color.CyanString(line)
			default:
				coloredLine = color.YellowString(line)
			}
		} else {
			coloredLine = color.YellowString(line)
		}

		fmt.Fprintln(w.Writer, coloredLine)
	}

	return len(p), scanner.Err()
}

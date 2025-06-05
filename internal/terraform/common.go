package terraform

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pterm/pterm"
)

// getTerraform locates the Terraform binary and initializes a Terraform instance
func GetTerraform(dir string) (*tfexec.Terraform, error) {
	terraformBinary, err := exec.LookPath("terraform")
	if err != nil {
		pterm.Error.Println("Terraform binary not found in PATH. Please install Terraform.")
		return nil, err
	}

	// Use the specified directory or default to current directory
	workingDir := "."
	if dir != "" {
		workingDir = dir
	}

	tf, err := tfexec.NewTerraform(workingDir, terraformBinary)
	if err != nil {
		pterm.Error.Printf("Error creating Terraform instance: %v\n", err)
		return nil, err
	}

	return tf, nil
}

// CustomColorWriter is a custom io.Writer that colors lines based on their prefix
// + for additions, - for deletions, ~ for changes, and no prefix for unchanged lines
// It also skips empty lines
func (w *CustomColorWriter) Write(p []byte) (n int, err error) {
	w.Buffer.Write(p)

	scanner := bufio.NewScanner(bytes.NewReader(p))

	for scanner.Scan() {
		line := scanner.Text()

		if len(strings.TrimSpace(line)) == 0 {
			fmt.Fprintln(w.Writer)
			continue
		}

		var coloredLine string

		trimmedLine := strings.TrimLeft(line, " ")
		if len(trimmedLine) > 0 {
			switch trimmedLine[0] {
			case '+':
				coloredLine = pterm.Green(line)
			case '-':
				coloredLine = pterm.Red(line)
			case '~':
				coloredLine = pterm.Cyan(line)
			default:
				coloredLine = pterm.Yellow(line)
			}
		} else {
			coloredLine = pterm.Yellow(line)
		}

		fmt.Fprintln(w.Writer, coloredLine)
	}

	return len(p), scanner.Err()
}

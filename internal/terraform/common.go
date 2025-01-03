package terraform

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pterm/pterm"
)

// getTerraform locates the Terraform binary and initializes a Terraform instance
func getTerraform() (*tfexec.Terraform, error) {
	terraformBinary, err := exec.LookPath("terraform")
	if err != nil {
		pterm.Error.Println("Terraform binary not found in PATH. Please install Terraform.")
		return nil, err
	}

	tf, err := tfexec.NewTerraform(".", terraformBinary)
	if err != nil {
		pterm.Error.Printf("Error creating Terraform instance: %v\n", err)
		return nil, err
	}

	pterm.Success.Printf("Configurations starting...\n")
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

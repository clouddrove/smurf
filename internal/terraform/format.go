package terraform

import (
	"os/exec"
	"bytes"

	"github.com/pterm/pterm"
)

// Format applies a canonical format to Terraform configuration files.
// It runs `terraform fmt` in the current directory to ensure that all
// Terraform files adhere to the standard formatting conventions.
func Format() error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	pterm.Info.Println("Formatting Terraform configuration files...")
	spinner, _ := pterm.DefaultSpinner.Start("Running terraform fmt")

	listCmd := exec.Command(tf.ExecPath(), "fmt", "-list=true")
	listCmd.Dir = "."

	fileList, err := listCmd.Output()
	if err != nil {
		var errOutput string
		if exitErr, ok := err.(*exec.ExitError); ok {
			errOutput = string(exitErr.Stderr)
		}
		spinner.Fail("Terraform format failed")
		pterm.Error.Printf("Terraform format failed: %v\n", err)
		if errOutput != "" {
			pterm.Error.Printf("Output: %s\n", errOutput)
		}
		return err
	}

	cmd := exec.Command(tf.ExecPath(), "fmt")
	cmd.Dir = "."

	if output, err := cmd.CombinedOutput(); err != nil {
		spinner.Fail("Terraform format failed")
		pterm.Error.Printf("Terraform format failed: %v\n", err)
		if len(output) > 0 {
			pterm.Error.Printf("Output: %s\n", string(output))
		}
		return err
	}

	spinner.Success("Terraform configuration files formatted successfully")

	if len(fileList) > 0 {
		pterm.Info.Println("\nFormatted files:")
		for _, file := range bytes.Split(fileList, []byte("\n")) {
			if len(file) > 0 {
				pterm.Info.Printf("- %s\n", string(file))
			}
		}
	} else {
		pterm.Info.Println("No files needed formatting")
	}

	return nil
}
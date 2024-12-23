package terraform

import (
	"os/exec"

	"github.com/pterm/pterm"
)

// Format applies a canonical format to Terraform configuration files
func Format() error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	pterm.Info.Println("Formatting Terraform configuration files...")
	spinner, _ := pterm.DefaultSpinner.Start("Running terraform fmt")

	cmd := exec.Command(tf.ExecPath(), "fmt")

	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		spinner.Fail("Terraform format failed")
		pterm.Error.Printf("Terraform format failed: %v\nOutput: %s\n", err, string(output))
		return err
	}
	spinner.Success("Terraform configuration files formatted successfully")

	return nil
}

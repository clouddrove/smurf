package terraform

import (
	"os/exec"

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

	pterm.Success.Printf("Using Terraform binary at: %s\n", terraformBinary)
	return tf, nil
}

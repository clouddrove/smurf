package terraform

import (
	"context"
	"os"

	"github.com/clouddrove/smurf/configs"
	"github.com/pterm/pterm"
)

// Validate checks the validity of the Terraform configuration
func Validate() error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	pterm.Info.Println("Validating Terraform configuration...")
	spinner, _ := pterm.DefaultSpinner.Start("Running terraform validate")

	valid, err := tf.Validate(context.Background())
	if err != nil {
		spinner.Fail("Terraform validation failed")
		pterm.Error.Printf("Terraform validation failed: %v\n", err)
		return err
	}

	customWriter := &configs.CustomColorWriter{Writer: os.Stdout}

	tf.SetStdout(customWriter)
	tf.SetStderr(os.Stderr)

	if valid.Valid {
		spinner.Success("Terraform configuration is valid.")
	} else {
		spinner.Fail("Terraform configuration is invalid.")
	}

	return nil
}

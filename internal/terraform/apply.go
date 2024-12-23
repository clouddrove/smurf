package terraform

import (
	"context"
	"os"

	"github.com/clouddrove/smurf/configs"
	"github.com/pterm/pterm"
)

// Apply executes 'terraform apply' to apply the planned changes
func Apply() error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	pterm.Info.Println("Applying Terraform changes...")
	spinner, _ := pterm.DefaultSpinner.Start("Running terraform apply")
	err = tf.Apply(context.Background())
	if err != nil {
		spinner.Fail("Terraform apply failed")
		pterm.Error.Printf("Terraform apply failed: %v\n", err)
		return err
	}

	customWriter := &configs.CustomColorWriter{Writer: os.Stdout}

	tf.SetStdout(customWriter)
	tf.SetStderr(os.Stderr)

	spinner.Success("Terraform applied successfully.")

	return nil
}

package terraform

import (
	"context"
	"os"

	"github.com/pterm/pterm"
)

// Apply executes 'apply' to apply the planned changes.
// It initializes the Terraform client, runs the apply operation with a spinner for user feedback,
// and handles any errors that occur during the process. Upon successful completion,
// it sets custom writers for stdout and stderr to handle colored output.
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

	customWriter := &CustomColorWriter{Writer: os.Stdout}

	tf.SetStdout(customWriter)
	tf.SetStderr(os.Stderr)

	spinner.Success("Terraform applied successfully.")

	return nil
}

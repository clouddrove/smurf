package terraform

import (
	"context"
	"os"

	"github.com/pterm/pterm"
)

// Destroy executes 'destroy' to remove all managed infrastructure.
// It initializes the Terraform client, sets up custom writers for colored output,
// runs the destroy operation with a spinner for user feedback, and handles any
// errors that occur during the process. Upon successful completion, it stops
// the spinner with a success message.
func Destroy() error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	customWriter := &CustomColorWriter{Writer: os.Stdout}

	tf.SetStdout(customWriter)
	tf.SetStderr(os.Stderr)

	pterm.Info.Println("Destroying Terraform resources...")
	spinner, _ := pterm.DefaultSpinner.Start("Running terraform destroy")
	err = tf.Destroy(context.Background())
	if err != nil {
		spinner.Fail("Terraform destroy failed")
		pterm.Error.Printf("Terraform destroy failed: %v\n", err)
		return err
	}
	spinner.Success("Terraform resources destroyed successfully.")

	return nil
}

package terraform

import (
	"context"
	"os"

	"github.com/pterm/pterm"
)

// Validate checks the validity of the Terraform configuration by running 'validate'.
// It provides user feedback through spinners and colored messages, and handles any errors
// that occur during the validation process.
func Validate() error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	pterm.Info.Println("Validating Infrastructure configuration...")
	spinner, _ := pterm.DefaultSpinner.Start("Running validation")

	valid, err := tf.Validate(context.Background())
	if err != nil {
		spinner.Fail("Infrastructure validation failed")
		pterm.Error.Printf("Infrastructure validation failed: %v\n", err)
		return err
	}

	customWriter := &CustomColorWriter{Writer: os.Stdout}

	tf.SetStdout(customWriter)
	tf.SetStderr(os.Stderr)

	if valid.Valid {
		spinner.Success("Infrastructure configuration is valid.")
	} else {
		spinner.Fail("Infrastructure configuration is invalid.")
	}

	return nil
}

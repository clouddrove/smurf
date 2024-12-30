package terraform

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pterm/pterm"
)

// Init initializes the Terraform working directory by running 'init'.
// It sets up the Terraform client, executes the initialization with upgrade options,
// and provides user feedback through spinners and colored messages.
// Upon successful initialization, it configures custom writers for enhanced output.
func Init() error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	pterm.Info.Println("Initializing Terraform...")
	spinner, _ := pterm.DefaultSpinner.Start("Running terraform init")
	err = tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		spinner.Fail("Terraform init failed")
		pterm.Error.Printf("Terraform init failed: %v\n", err)
		return err
	}

	customWriter := &CustomColorWriter{Writer: os.Stdout}

	tf.SetStdout(customWriter)
	tf.SetStderr(os.Stderr)

	spinner.Success("Terraform initialized successfully")

	pterm.Success.Println("Terraform configuration validated successfully.")
	return nil
}

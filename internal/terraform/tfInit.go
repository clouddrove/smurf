package terraform

import (
	"context"
	"os"

	"github.com/clouddrove/smurf/configs"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pterm/pterm"
)

// Init initializes Terraform
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

	customWriter := &configs.CustomColorWriter{Writer: os.Stdout}

	tf.SetStdout(customWriter)
	tf.SetStderr(os.Stderr)

	spinner.Success("Terraform initialized successfully")

	pterm.Success.Println("Terraform configuration validated successfully.")
	return nil
}

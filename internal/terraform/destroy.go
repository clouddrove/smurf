package terraform

import (
	"context"
	"os"

	"github.com/clouddrove/smurf/configs"
	"github.com/pterm/pterm"
)

// Destroy removes all resources managed by Terraform
func Destroy() error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	customWriter := &configs.CustomColorWriter{Writer: os.Stdout}

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

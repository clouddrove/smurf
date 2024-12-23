package terraform

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
)

// Output displays the outputs defined in the Terraform configuration
func Output() error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	tf.SetStdout(os.Stdout)
	tf.SetStderr(os.Stderr)

	pterm.Info.Println("Refreshing Terraform state...")
	spinner, _ := pterm.DefaultSpinner.Start("Running terraform refresh")
	err = tf.Refresh(context.Background())
	if err != nil {
		spinner.Fail("Error refreshing Terraform state")
		pterm.Error.Printf("Error refreshing Terraform state: %v\n", err)
		return err
	}
	spinner.Success("Terraform state refreshed successfully.")

	outputs, err := tf.Output(context.Background())
	if err != nil {
		pterm.Error.Printf("Error getting Terraform outputs: %v\n", err)
		return err
	}

	if len(outputs) == 0 {
		pterm.Info.Println("No outputs found.")
		return nil
	}

	green := color.New(color.FgGreen).SprintfFunc()

	pterm.Info.Println("Terraform outputs:")
	for key, value := range outputs {
		if value.Sensitive {
			fmt.Println(green("%s: [sensitive value hidden]", key))
		} else {
			fmt.Println(green("%s: %v", key, value.Value))
		}
	}

	return nil
}

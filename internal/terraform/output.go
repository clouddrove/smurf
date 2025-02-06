package terraform

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
)

// Output displays the outputs defined in the Terraform configuration.
// It refreshes the Terraform state to ensure it reflects the current infrastructure,
// then retrieves and displays the outputs. Sensitive outputs are hidden for security.
// Provides user feedback through spinners and colored messages for enhanced UX.
func Output() error {
	tf, err := GetTerraform()
	if err != nil {
		return err
	}

	tf.SetStdout(os.Stdout)
	tf.SetStderr(os.Stderr)

	pterm.Info.Println("Refreshing Infrastructure state...")
	spinner, _ := pterm.DefaultSpinner.Start("Refreshing Infrastructure state...")
	err = tf.Refresh(context.Background())
	if err != nil {
		spinner.Fail("Error refreshing  state")
		pterm.Error.Printf("Error refreshing  state: %v\n", err)
		return err
	}
	spinner.Success("State refreshed successfully.")

	outputs, err := tf.Output(context.Background())
	if err != nil {
		pterm.Error.Printf("Error getting Infrastructure outputs: %v\n", err)
		return err
	}

	if len(outputs) == 0 {
		pterm.Info.Println("No outputs found.")
		return nil
	}

	green := color.New(color.FgGreen).SprintfFunc()

	pterm.Info.Println("Outputs: ")
	for key, value := range outputs {
		if value.Sensitive {
			fmt.Println(green("%s: [sensitive value hidden]", key))
		} else {
			fmt.Println(green("%s: %v", key, value.Value))
		}
	}

	return nil
}

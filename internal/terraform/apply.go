package terraform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pterm/pterm"
)

// Apply executes 'apply' to apply the planned changes.
// It initializes the Terraform client, runs the apply operation with a spinner for user feedback,
// and handles any errors that occur during the process. Upon successful completion,
// it sets custom writers for stdout and stderr to handle colored output.
func Apply(approve bool) error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgBlue)).
		WithTextStyle(pterm.NewStyle(pterm.FgLightWhite)).
		Printf("Terraform Apply")
	fmt.Println()

	planOutput, err := tf.Plan(context.Background())
	if err != nil {
		pterm.Error.Printf("Failed to generate plan: %v\n", err)
		return err
	}

	if !planOutput {
		pterm.Info.Println("No changes to apply.")
		return nil
	}

	_, err = tf.Plan(context.Background(), tfexec.Out("plan.out"))
	if err != nil {
		pterm.Error.Printf("Failed to generate plan: %v\n", err)
		return err
	}

	show, err := tf.ShowPlanFile(context.Background(), "plan.out")
	if err != nil {
		pterm.Error.Printf("Failed to show plan: %v\n", err)
		return err
	}

	if len(show.ResourceChanges) > 0 {
		pterm.Info.Println("\nPlanned Changes:")
		for _, resource := range show.ResourceChanges {
			action := strings.ToUpper(string(resource.Change.Actions[0]))
			switch action {
			case "CREATE":
				color.Green("  + %s\n", resource.Address)
			case "UPDATE":
				color.Yellow("  ~ %s\n", resource.Address)
			case "DELETE":
				color.Red("  - %s\n", resource.Address)
			}
		}
		fmt.Println()
	}

	if !approve {
		pterm.Warning.Println("Apply cancelled. Use --approve to approve changes.")
		return nil
	}

	spinner := pterm.DefaultSpinner.
		WithRemoveWhenDone(true).
		WithText("Applying changes...")
	spinner.Start()

	var outputBuffer bytes.Buffer
	customWriter := &CustomColorWriter{
		Buffer: &outputBuffer,
		Writer: os.Stdout,
	}

	tf.SetStdout(customWriter)
	tf.SetStderr(os.Stderr)

	err = tf.Apply(context.Background())
	if err != nil {
		spinner.Fail("Apply failed")
		pterm.Error.Printf("Error: %v\n", err)
		return err
	}

	spinner.Success("Applied successfully")

	added := 0
	changed := 0
	destroyed := 0

	for _, resource := range show.ResourceChanges {
		for _, action := range resource.Change.Actions {
			switch strings.ToUpper(string(action)) {
			case "CREATE":
				added++
			case "UPDATE":
				changed++
			case "DELETE":
				destroyed++
			}
		}
	}

	pterm.Success.Println("\nApply complete! Resources: " +
		color.GreenString(fmt.Sprintf("%d added", added)) +
		color.YellowString(fmt.Sprintf(", %d changed", changed)) +
		color.RedString(fmt.Sprintf(", %d destroyed", destroyed)))

	return nil
}

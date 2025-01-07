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

	_, err = tf.Plan(context.Background(), tfexec.Out("plan.out"))
	if err != nil {
		pterm.Error.Printf("Failed to generate plan: %v\n", err)
		return err
	}

	planDetail, err := tf.ShowPlanFileRaw(context.Background(), "plan.out")
	if err != nil {
		pterm.Error.Printf("Failed to show plan: %v\n", err)
		return err
	}

	var outputBuffer bytes.Buffer
	customWriter := &CustomColorWriter{
		Buffer: &outputBuffer,
		Writer: os.Stdout,
	}
	customWriter.Write([]byte(planDetail))

	show, err := tf.ShowPlanFile(context.Background(), "plan.out")
	if err != nil {
		pterm.Error.Printf("Failed to parse plan: %v\n", err)
		return err
	}

	if len(show.ResourceChanges) == 0 {
		pterm.Info.Println("No changes to apply.")
		return nil
	}

	if !approve {
		var confirmation string
		fmt.Print("\nDo you want to perform these actions? Only 'yes' will be accepted to approve.\nEnter a value: ")
		fmt.Scanln(&confirmation)

		if confirmation != "yes" {
			return nil
		}
	}

	spinner := pterm.DefaultSpinner.
		WithRemoveWhenDone(true).
		WithText("Applying changes...")
	spinner.Start()

	
	tf.SetStdout(os.Stdout)
	tf.SetStderr(os.Stderr)

	err = tf.Apply(context.Background(), tfexec.DirOrPlan("plan.out"))
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
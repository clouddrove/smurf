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

// Destroy executes 'destroy' to remove all managed infrastructure.
// It initializes the Terraform client, sets up custom writers for colored output,
// runs the destroy operation with a spinner for user feedback, and handles any
// errors that occur during the process. Upon successful completion, it stops
// the spinner with a success message.
func Destroy(approve bool) error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	pterm.DefaultHeader.
		WithBackgroundStyle(pterm.NewStyle(pterm.BgRed)).
		WithTextStyle(pterm.NewStyle(pterm.FgLightWhite)).
		Printf("Terraform Destroy")
	fmt.Println()

	_, err = tf.Plan(
		context.Background(),
		tfexec.Destroy(true),
		tfexec.Out("plan.out"),
	)
	if err != nil {
		pterm.Error.Printf("Failed to generate destroy plan: %v\n", err)
		return err
	}

	show, err := tf.ShowPlanFile(context.Background(), "plan.out")
	if err != nil {
		pterm.Error.Printf("Failed to parse plan: %v\n", err)
		return err
	}

	if len(show.ResourceChanges) == 0 {
		pterm.Info.Println("No resources to destroy.")
		return nil
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
	if !approve {
		var confirmation string
		fmt.Print("\nDo you want to destroy these resources? Only 'yes' will be accepted to approve.\nEnter a value: ")
		fmt.Scanln(&confirmation)
		fmt.Println()

		if confirmation != "yes" {
			return nil
		}
	}

	spinner := pterm.DefaultSpinner.
		WithRemoveWhenDone(true).
		WithStyle(pterm.NewStyle(pterm.FgLightRed)).
		WithText("Destroying resources...")
	spinner.Start()

	tf.SetStdout(os.Stdout)
	tf.SetStderr(os.Stderr)

	err = tf.Apply(
		context.Background(),
		tfexec.Destroy(true),
		tfexec.DirOrPlan("plan.out"),
	)
	if err != nil {
		spinner.Fail("Destroy failed")
		pterm.Error.Printf("Error: %v\n", err)
		return err
	}

	spinner.Success("Destroyed successfully")

	destroyed := 0
	for _, rc := range show.ResourceChanges {
		for _, action := range rc.Change.Actions {
			if strings.ToUpper(string(action)) == "DELETE" {
				destroyed++
			}
		}
	}

	pterm.Success.Println(
		"\nDestroy complete! Resources: " +
			color.RedString("%d destroyed", destroyed),
	)

	return nil
}

// DestroyLogger extended to handle destroy-specific output
type DestroyLogger struct {
	CustomColorWriter
	isDestroy bool
}

// Write handles the output of the Terraform destroy command
// and applies color to specific messages
func (l *DestroyLogger) Write(p []byte) (n int, err error) {
	msg := string(p)

	if l.isDestroy {
		if strings.Contains(msg, "Destroying...") {
			msg = color.RedString(msg)
		} else if strings.Contains(msg, "Destruction complete") {
			msg = color.GreenString(msg)
		}
	}

	switch {
	case strings.Contains(msg, "Terraform will perform the following actions"):
		pterm.Info.Println(msg)
	case strings.Contains(msg, "Plan:"):
		color.Yellow(msg)
	case strings.Contains(msg, "Error:"):
		color.Red(msg)
	default:
		if _, err := l.CustomColorWriter.Writer.Write([]byte(msg)); err != nil {
			return 0, err
		}
	}

	return len(p), nil
}

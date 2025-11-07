package terraform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-exec/tfexec"
)

// Destroy executes 'destroy' to remove all managed infrastructure.
// It initializes the Terraform client, sets up custom writers for colored output,
// runs the destroy operation with a spinner for user feedback, and handles any
// errors that occur during the process. Upon successful completion, it stops
// the spinner with a success message.
func Destroy(approve bool, lock bool, dir string) error {
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform client: %v", err)
		return err
	}

	Info("Preparing Terraform destroy operation in directory: %s", dir)

	// Generate destroy plan
	_, err = tf.Plan(
		context.Background(),
		tfexec.Destroy(true),
		tfexec.Out("plan.out"),
	)
	if err != nil {
		Error("Failed to generate destroy plan: %v", err)
		return err
	}

	show, err := tf.ShowPlanFile(context.Background(), "plan.out")
	if err != nil {
		Error("Failed to parse plan: %v", err)
		return err
	}

	if len(show.ResourceChanges) == 0 {
		Warn("No resources to destroy.")
		return nil
	}

	planDetail, err := tf.ShowPlanFileRaw(context.Background(), "plan.out")
	if err != nil {
		Error("Failed to show plan details: %v", err)
		return err
	}

	var outputBuffer bytes.Buffer
	customWriter := &CustomColorWriter{
		Buffer: &outputBuffer,
		Writer: os.Stdout,
	}
	customWriter.Write([]byte(planDetail))

	// Ask for confirmation if not auto-approved
	if !approve {
		var confirmation string
		fmt.Print("\nDo you want to destroy these resources? Only 'yes' will be accepted to approve.\nEnter a value: ")
		fmt.Scanln(&confirmation)
		fmt.Println()

		if confirmation != "yes" {
			Warn("Destroy operation aborted by user.")
			return nil
		}
	}

	tf.SetStdout(os.Stdout)
	tf.SetStderr(os.Stderr)

	err = tf.Apply(
		context.Background(),
		tfexec.Destroy(true),
		tfexec.DirOrPlan("plan.out"),
		tfexec.Lock(lock),
	)
	if err != nil {
		Error("Terraform destroy failed: %v", err)
		return err
	}

	destroyed := 0
	for _, rc := range show.ResourceChanges {
		for _, action := range rc.Change.Actions {
			if strings.ToUpper(string(action)) == "DELETE" {
				destroyed++
			}
		}
	}

	Success("Destroy complete! Resources destroyed: %d", destroyed)
	return nil
}

// DestroyLogger extended to handle destroy-specific output
type DestroyLogger struct {
	CustomColorWriter
	isDestroy bool
}

// Write handles the output of the Terraform destroy command
// and applies color to specific messages
// Write enhances Terraform destroy logs with color and readability.
func (l *DestroyLogger) Write(p []byte) (n int, err error) {
	msg := string(p)

	if l.isDestroy {
		switch {
		case strings.Contains(msg, "Destroying..."):
			Warn("Destroying: %s", strings.TrimSpace(msg))
		case strings.Contains(msg, "Destruction complete"):
			Success("Destruction complete: %s", strings.TrimSpace(msg))
		case strings.Contains(msg, "Error:"):
			Error("%s", strings.TrimSpace(msg))
		default:
			fmt.Print(msg)
		}
		return len(p), nil
	}

	return l.CustomColorWriter.Writer.Write(p)
}

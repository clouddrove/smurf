package terraform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-exec/tfexec"
)

// Apply executes 'apply' to apply the planned changes.
// It initializes the Terraform client, runs the apply operation with a spinner for user feedback,
// and handles any errors that occur during the process. Upon successful completion,
// it sets custom writers for stdout and stderr to handle colored output. lock is by default false
func Apply(approve bool, vars []string, varFiles []string, lock bool, dir string) error {
	Step("Initializing Terraform client...")
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform client: %v", err)
		return err
	}

	applyOptions := []tfexec.PlanOption{
		tfexec.Out("plan.out"),
	}

	// Handle inline variables
	if vars != nil {
		Info("Setting %d variable(s)...", len(vars))
		for _, v := range vars {
			Info("Using variable: %s", v)
			applyOptions = append(applyOptions, tfexec.Var(v))
		}
	}

	// Handle variable files
	if varFiles != nil {
		Info("Loading %d variable file(s)...", len(varFiles))
		for _, vf := range varFiles {
			Info("Using var-file: %s", vf)
			applyOptions = append(applyOptions, tfexec.VarFile(vf))
		}
	}

	// Generate Terraform plan
	Step("Generating Terraform plan...")
	_, err = tf.Plan(context.Background(), applyOptions...)
	if err != nil {
		Error("Failed to generate plan: %v", err)
		return err
	}
	Success("Terraform plan generated successfully.")

	// Show plan details
	Step("Fetching plan details...")
	planDetail, err := tf.ShowPlanFileRaw(context.Background(), "plan.out")
	if err != nil {
		Error("Failed to read plan details: %v", err)
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
		Error("Failed to parse plan file: %v", err)
		return err
	}

	if len(show.ResourceChanges) == 0 {
		Warn("No changes to apply. Everything is up to date.")
		return nil
	}

	// Approval prompt (if not auto-approved)
	if !approve {
		var confirmation string
		fmt.Print("\nDo you want to perform these actions? Only 'yes' will be accepted to approve.\nEnter a value: ")
		fmt.Scanln(&confirmation)
		fmt.Println()

		if confirmation != "yes" {
			Warn("Operation cancelled by user.")
			return nil
		}
	}

	// Apply phase with spinner
	Step("Applying changes...")

	tf.SetStdout(os.Stdout)
	tf.SetStderr(os.Stderr)

	Options := []tfexec.ApplyOption{
		tfexec.Lock(lock),
		tfexec.DirOrPlan("plan.out"),
	}

	err = tf.Apply(context.Background(), Options...)
	if err != nil {
		Error("Terraform apply failed: %v", err)
		return err
	}

	Success("Terraform changes applied successfully.")

	// Summarize resource changes
	added, changed, destroyed := 0, 0, 0

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

	Success("Apply complete! Resources: %d added, %d changed, %d destroyed", added, changed, destroyed)
	return nil
}

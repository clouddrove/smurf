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
func Apply(approve bool, vars []string, varFiles []string, lock bool, dir string, targets []string, state string) error {
	defer cleanupPlanFile()

	Step("Initializing Terraform client...")
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform client: %v", err)
		return err
	}

	applyOptions := []tfexec.PlanOption{
		tfexec.Out("plan.out"),
	}

	// Handle state file
	if state != "" {
		Info("Using custom state file: %s", state)
		applyOptions = append(applyOptions, tfexec.State(state))
	}

	// Handle inline variables
	if vars != nil {
		Info("Setting %d variable(s)...", len(vars))
		for _, v := range vars {
			Info("Using variable: %s", v)
			applyOptions = append(applyOptions, tfexec.Var(v))
		}
	}

	// Handle variable files with existence check
	if varFiles != nil {
		Info("Loading %d variable file(s)...", len(varFiles))
		for _, vf := range varFiles {
			if _, err := os.Stat(vf); os.IsNotExist(err) {
				Error("Variable file not found: %s", vf)
				return fmt.Errorf("variable file not found: %s", vf)
			}
			Info("Using var-file: %s", vf)
			applyOptions = append(applyOptions, tfexec.VarFile(vf))
		}
	}

	// Handle targets
	if targets != nil {
		Info("Targeting %d resource(s)...", len(targets))
		for _, target := range targets {
			Info("Using target: %s", target)
			applyOptions = append(applyOptions, tfexec.Target(target))
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

	// NEW: Colorize the "No changes" message if present
	planStr := string(planDetail)
	if strings.Contains(planStr, "No changes.") {
		// Find and colorize the "No changes" section
		lines := strings.Split(planStr, "\n")
		for i, line := range lines {
			if strings.Contains(line, "No changes.") ||
				strings.Contains(line, "Your infrastructure matches the configuration") ||
				strings.Contains(line, "found no differences") ||
				strings.Contains(line, "so no changes are needed") {
				// Colorize these lines in green (like Terraform CLI does)
				lines[i] = fmt.Sprintf("\033[32m%s\033[0m", line)
			}
		}
		planStr = strings.Join(lines, "\n")
	}

	customWriter.Write([]byte(planStr))

	show, err := tf.ShowPlanFile(context.Background(), "plan.out")
	if err != nil {
		Error("Failed to parse plan file: %v", err)
		return err
	}

	if len(show.ResourceChanges) == 0 {
		// Also colorize the warning message when no changes are found
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

	applyOpts := []tfexec.ApplyOption{
		tfexec.Lock(lock),
		tfexec.DirOrPlan("plan.out"),
	}

	// Add state option to apply as well
	if state != "" {
		applyOpts = append(applyOpts, tfexec.State(state))
	}

	// Add target options to apply as well
	if len(targets) > 0 {
		for _, target := range targets {
			applyOpts = append(applyOpts, tfexec.Target(target))
		}
	}

	err = tf.Apply(context.Background(), applyOpts...)
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

// cleanupPlanFile removes the temporary plan file
func cleanupPlanFile() {
	if _, err := os.Stat("plan.out"); err == nil {
		os.Remove("plan.out")
	}
}

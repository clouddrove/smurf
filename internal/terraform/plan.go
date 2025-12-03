package terraform

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-exec/tfexec"
)

// Plan runs 'terraform plan' and outputs the plan to the console.
// It allows setting variables either via command-line arguments or variable files.
// The function provides user feedback through spinners and colored messages,
// and handles any errors that occur during the planning process.
func Plan(vars []string, varFiles []string, dir string, destroy bool, targets []string, refresh bool, state string, planOut string) error {
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform client: %v", err)
		return err
	}

	var outputBuffer bytes.Buffer
	customWriter := &CustomColorWriter{
		Buffer: &outputBuffer,
		Writer: os.Stdout,
	}

	tf.SetStdout(customWriter)
	tf.SetStderr(os.Stderr)

	// Start planning process
	Info("Starting infrastructure planning in directory: %s", dir)

	planOptions := []tfexec.PlanOption{}

	// Handle plan output file
	if planOut != "" {
		// Ensure the plan file path is absolute or relative to the current directory
		if !filepath.IsAbs(planOut) {
			planOut = filepath.Join(dir, planOut)
		}
		Info("Saving plan to: %s", planOut)
		planOptions = append(planOptions, tfexec.Out(planOut))
	}

	// Handle state file - add this block
	if state != "" {
		Info("Using custom state file: %s", state)
		planOptions = append(planOptions, tfexec.State(state))
	}

	// Apply variables
	if len(vars) > 0 {
		for _, v := range vars {
			Info("Applying variable: %s", v)
			planOptions = append(planOptions, tfexec.Var(v))
		}
	}

	// Apply variable files
	if len(varFiles) > 0 {
		for _, vf := range varFiles {
			Info("Loading variable file: %s", vf)
			planOptions = append(planOptions, tfexec.VarFile(vf))
		}
	}

	// Handle targets
	if len(targets) > 0 {
		Info("Targeting %d resource(s)...", len(targets))
		for _, target := range targets {
			Info("Using target: %s", target)
			planOptions = append(planOptions, tfexec.Target(target))
		}
	}

	// Destroy flag support
	if destroy {
		Warn("Planning for destruction of infrastructure resources...")
		planOptions = append(planOptions, tfexec.Destroy(true))
	}

	// Refresh flag support
	if !refresh {
		Info("Skipping state refresh...")
		planOptions = append(planOptions, tfexec.Refresh(false))
	}

	// Execute Terraform plan
	hasChanges, err := tf.Plan(context.Background(), planOptions...)
	if err != nil {
		Error("Terraform plan failed: %v", err)
		return err
	}

	// Provide summary based on plan output
	if planOut != "" {
		if hasChanges {
			Success("Terraform plan executed successfully with changes.")
			Info("Plan saved to: %s", planOut)
			Info("To apply this plan, run: smurf stf apply %s", filepath.Base(planOut))
		} else {
			Success("No changes. Infrastructure is up-to-date.")
			Info("Plan saved to: %s", planOut)
		}
	} else {
		if hasChanges {
			Success("Terraform plan executed successfully with changes.")
			Info("Review the changes above before applying.")
		} else {
			Success("No changes. Infrastructure is up-to-date.")
		}
	}

	Success("Terraform plan executed successfully. Review the changes above before applying.")
	return nil
}

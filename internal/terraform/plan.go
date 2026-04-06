package terraform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/hashicorp/terraform-exec/tfexec"
)

// Plan runs 'terraform plan' and outputs the plan to the console.
// It allows setting variables either via command-line arguments or variable files.
// The function provides user feedback through spinners and colored messages,
// and handles any errors that occur during the planning process.
func Plan(vars []string, varFiles []string,
	dir string, destroy bool,
	targets []string, refresh bool,
	state string, out string,
	useAI bool) error {

	Step("Initializing Terraform client...")
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform client: %v", err)
		ai.AIExplainError(useAI, err.Error())
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

	// Handle state file
	if state != "" {
		Info("Using custom state file: %s", state)
		planOptions = append(planOptions, tfexec.State(state))
	}

	// Handle output plan file
	if out != "" {
		// Validate output path
		outDir := filepath.Dir(out)
		if outDir != "" && outDir != "." {
			if _, err := os.Stat(outDir); os.IsNotExist(err) {
				Error("Output directory does not exist: %s", outDir)
				return fmt.Errorf("output directory does not exist: %s", outDir)
			}
		}

		// Check if we can write to the file
		if _, err := os.Stat(out); err == nil {
			Warn("Plan file %s already exists and will be overwritten", out)
		}

		Info("Saving execution plan to: %s", out)
		planOptions = append(planOptions, tfexec.Out(out))
	}

	// Apply variables
	if len(vars) > 0 {
		Info("Setting %d variable(s)...", len(vars))
		for _, v := range vars {
			Info("Using variable: %s", v)
			planOptions = append(planOptions, tfexec.Var(v))
		}
	}

	// Apply variable files with validation
	if len(varFiles) > 0 {
		Info("Loading %d variable file(s)...", len(varFiles))
		for _, vf := range varFiles {
			if _, err := os.Stat(vf); os.IsNotExist(err) {
				Error("Variable file not found: %s", vf)
				ai.AIExplainError(useAI, fmt.Sprintf("Variable file not found: %s", vf))
				return fmt.Errorf("variable file not found: %s", vf)
			}
			Info("Using var-file: %s", vf)
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

	// Execute Terraform plan and get the hasChanges boolean
	Step("Generating Terraform plan...")
	hasChanges, err := tf.Plan(context.Background(), planOptions...)
	if err != nil {
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	// Handle based on whether changes were detected
	if hasChanges {
		if out != "" {
			Success(" Terraform plan saved to: %s", out)
			Info("To apply this plan, run: smurf stf apply %s", out)
		} else {
			Success("\nTerraform plan executed successfully. Review the changes above before applying.")
		}
	} else {
		Success("\nNo changes. Your infrastructure matches the configuration.")

		if out != "" {
			// If we saved a plan file but there are no changes, it's still saved but empty
			Info("Note: Plan file was saved even though no changes detected: %s", out)
			// Clean up empty plan file
			if fileInfo, err := os.Stat(out); err == nil && fileInfo.Size() == 0 {
				os.Remove(out)
				Info("Removed empty plan file: %s", out)
			}
		}
	}

	return nil
}

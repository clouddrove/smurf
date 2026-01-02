package terraform

import (
	"context"
	"fmt"
	"os"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/hashicorp/terraform-exec/tfexec"
)

// DetectDrift checks for drift between the Terraform state and the actual infrastructure.
// It performs a `plan` with refresh enabled to detect any changes that differ
// from the current state. If drift is detected, it lists the affected resources.
// Provides user feedback through spinners and consistent Smurf log style.
func DetectDrift(dir string, useAI bool) error {
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	planFile := "drift.plan"

	Info("Starting Terraform drift detection...")

	// Generate drift plan
	_, err = tf.Plan(context.Background(), tfexec.Out(planFile), tfexec.Refresh(true))
	if err != nil {
		Error("Failed to execute Terraform plan for drift detection: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	tf.SetStderr(os.Stderr)

	// Parse plan file
	plan, err := tf.ShowPlanFile(context.Background(), planFile)
	if err != nil {
		Error("Failed to read drift plan file: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	if len(plan.ResourceChanges) > 0 {
		Warn("Drift detected in your infrastructure:")
		for _, change := range plan.ResourceChanges {
			fmt.Printf("  - %s: %v\n", change.Address, change.Change.Actions)
		}
		Warn("Run 'terraform apply' to reconcile drifted resources.")
	} else {
		Success("No drift detected. Your infrastructure is in sync.")
	}

	return nil
}

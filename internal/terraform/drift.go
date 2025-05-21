package terraform

import (
	"context"
	"os"

	"github.com/fatih/color"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pterm/pterm"
)

// DetectDrift checks for drift between the Terraform state and the actual infrastructure.
// It performs a `plan` with refresh enabled to detect any changes that differ
// from the current state. If drift is detected, it lists the affected resources.
// Provides user feedback through spinners and colored messages for better UX.
func DetectDrift(dir string) error {
	tf, err := GetTerraform(dir)
	if err != nil {
		return err
	}

	planFile := "drift.plan"
	pterm.Info.Println("Checking for drift...")
	spinner, _ := pterm.DefaultSpinner.Start("Running terraform plan for drift detection")
	_, err = tf.Plan(context.Background(), tfexec.Out(planFile), tfexec.Refresh(true))

	if err != nil {
		spinner.Fail("Terraform plan for drift detection failed")
		pterm.Error.Printf("Terraform plan for drift detection failed: %v\n", err)
		return err
	}
	spinner.Success("Terraform drift detection plan completed")

	plan, err := tf.ShowPlanFile(context.Background(), planFile)
	if err != nil {
		pterm.Error.Printf("Error showing plan file: %v\n", err)
		return err
	}

	tf.SetStderr(os.Stderr)

	if len(plan.ResourceChanges) > 0 {
		pterm.Warning.Println("Drift detected:")
		for _, change := range plan.ResourceChanges {
			pterm.Print(color.YellowString("- %s: %s\n", change.Address, change.Change.Actions))
		}
	} else {
		pterm.Success.Println("No drift detected.")
	}

	return nil
}

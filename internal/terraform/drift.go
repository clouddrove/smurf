package terraform

import (
	"context"
	"os"

	"github.com/fatih/color"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pterm/pterm"
)

// DetectDrift checks for drift between the Terraform state and the actual infrastructure
func DetectDrift() error {
	tf, err := getTerraform()
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
			pterm.Printf(color.YellowString("- %s: %s\n", change.Address, change.Change.Actions))
		}
	} else {
		pterm.Success.Println("No drift detected.")
	}

	return nil
}
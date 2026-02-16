package terraform

import (
	"context"
	"fmt"
	"os"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/hashicorp/terraform-exec/tfexec"
)

func Plan(
	vars []string,
	varFiles []string,
	dir string,
	destroy bool,
	targets []string,
	refresh bool,
	state string,
	out string,
	jsonOutput bool,
	pushToCloud bool,
	useAI bool,
) error {

	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	tf.SetStdout(os.Stdout)
	tf.SetStderr(os.Stderr)

	planOptions := []tfexec.PlanOption{}

	if state != "" {
		planOptions = append(planOptions, tfexec.State(state))
	}

	if out != "" {
		planOptions = append(planOptions, tfexec.Out(out))
	}

	for _, v := range vars {
		planOptions = append(planOptions, tfexec.Var(v))
	}

	for _, vf := range varFiles {
		planOptions = append(planOptions, tfexec.VarFile(vf))
	}

	for _, target := range targets {
		planOptions = append(planOptions, tfexec.Target(target))
	}

	if destroy {
		planOptions = append(planOptions, tfexec.Destroy(true))
	}

	if !refresh {
		planOptions = append(planOptions, tfexec.Refresh(false))
	}

	// Execute plan
	hasChanges, err := tf.Plan(context.Background(), planOptions...)
	if err != nil {
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	if !hasChanges {
		Success("✔ No changes. Infrastructure is up-to-date.")
		return nil
	}

	Success("Plan completed successfully.")

	// -----------------------------
	// JSON Output Section
	// -----------------------------
	if jsonOutput {

		if out == "" {
			return fmt.Errorf("JSON output requires --out plan file")
		}

		Info("Generating JSON plan output...")

		planJSON, err := tf.ShowPlanFileRaw(context.Background(), out)
		if err != nil {
			return err
		}

		jsonFile := "plan.json"

		// FIXED HERE 👇
		err = os.WriteFile(jsonFile, []byte(planJSON), 0644)
		if err != nil {
			return err
		}

		Success("JSON plan saved to %s", jsonFile)
	}

	return nil
}

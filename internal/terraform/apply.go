package terraform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

func Apply(approve bool, vars []string,
	varFiles []string, lock bool,
	dir string, targets []string,
	state string, useAI bool) error {

	defer cleanupPlanFile()

	Step("Initializing Terraform client...")
	tf, err := initTerraform(dir, useAI)
	if err != nil {
		Error("Failed to initialize Terraform client: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	planOptions, err := buildPlanOptions(vars, varFiles, targets, state)
	if err != nil {
		Error("Failed to build plan: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	// Generate Plan
	Step("Generating Terraform plan...")
	_, err = tf.Plan(context.Background(), append(planOptions, tfexec.Out("plan.out"))...)
	if err != nil {
		Error("Failed to generate plan: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}
	Success("Terraform plan generated successfully.")

	// Show Plan
	show, err := showPlan(tf, "plan.out", useAI)
	if err != nil {
		return err
	}

	if len(show.ResourceChanges) == 0 {
		Warn("No changes to apply. Everything is up to date.")
		return nil
	}

	// Approval
	if !approve {
		if !askForApproval() {
			Warn("Operation cancelled by user.")
			return nil
		}
	}

	// Apply
	Step("Applying changes...")
	tf.SetStdout(os.Stdout)
	tf.SetStderr(os.Stderr)

	applyOpts := buildApplyOptions(lock, "plan.out", state, targets)

	err = tf.Apply(context.Background(), applyOpts...)
	if err != nil {
		Error("Terraform apply failed: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	Success("Terraform changes applied successfully.")
	summarizeChanges(show)

	return nil
}

func ApplyWithPlan(planFile string, vars []string,
	varFiles []string, lock bool,
	dir string, targets []string,
	state string, useAI bool) error {

	Step("Initializing Terraform client...")
	tf, err := initTerraform(dir, useAI)
	if err != nil {
		return err
	}

	if _, err := os.Stat(planFile); os.IsNotExist(err) {
		return fmt.Errorf("plan file not found: %s", planFile)
	}

	Info("Applying plan from file: %s", planFile)

	show, err := showPlan(tf, planFile, useAI)
	if err != nil {
		return err
	}

	if len(show.ResourceChanges) == 0 {
		Warn("No changes to apply. Everything is up to date.")
		return nil
	}

	Step("Applying changes from plan file...")
	tf.SetStdout(os.Stdout)
	tf.SetStderr(os.Stderr)

	applyOpts := buildApplyOptions(lock, planFile, state, targets)

	applyOpts, err = addVarsAndFiles(vars, varFiles, applyOpts)
	if err != nil {
		return err
	}

	err = tf.Apply(context.Background(), applyOpts...)
	if err != nil {
		Error("Terraform apply failed: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	Success("Terraform changes applied successfully.")
	summarizeChanges(show)

	return nil
}

///////////////////////
// 🔧 Helper Functions
///////////////////////

func initTerraform(dir string, useAI bool) (*tfexec.Terraform, error) {
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform client: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return nil, err
	}
	return tf, nil
}

func buildPlanOptions(vars []string, varFiles []string, targets []string, state string) ([]tfexec.PlanOption, error) {
	opts := []tfexec.PlanOption{}

	if state != "" {
		opts = append(opts, tfexec.State(state))
	}

	for _, v := range vars {
		opts = append(opts, tfexec.Var(v))
	}

	for _, vf := range varFiles {
		if _, err := os.Stat(vf); os.IsNotExist(err) {
			return nil, fmt.Errorf("variable file not found: %s", vf)
		}
		opts = append(opts, tfexec.VarFile(vf))
	}

	for _, t := range targets {
		opts = append(opts, tfexec.Target(t))
	}

	return opts, nil
}

func buildApplyOptions(lock bool, planOrDir string, state string, targets []string) []tfexec.ApplyOption {
	opts := []tfexec.ApplyOption{
		tfexec.Lock(lock),
		tfexec.DirOrPlan(planOrDir),
	}

	if state != "" {
		opts = append(opts, tfexec.State(state))
	}

	for _, t := range targets {
		opts = append(opts, tfexec.Target(t))
	}

	return opts
}

func addVarsAndFiles(vars []string, varFiles []string, opts []tfexec.ApplyOption) ([]tfexec.ApplyOption, error) {
	for _, vf := range varFiles {
		if _, err := os.Stat(vf); os.IsNotExist(err) {
			return nil, fmt.Errorf("variable file not found: %s", vf)
		}
		opts = append(opts, tfexec.VarFile(vf))
	}

	for _, v := range vars {
		opts = append(opts, tfexec.Var(v))
	}

	return opts, nil
}

func showPlan(tf *tfexec.Terraform, planFile string, useAI bool) (*tfjson.Plan, error) {
	Step("Showing plan details...")

	planDetail, err := tf.ShowPlanFileRaw(context.Background(), planFile)
	if err != nil {
		Error("Failed to read plan: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return nil, err
	}

	var buffer bytes.Buffer
	writer := &CustomColorWriter{
		Buffer: &buffer,
		Writer: os.Stdout,
	}

	writer.Write([]byte(colorizeNoChanges(string(planDetail))))

	show, err := tf.ShowPlanFile(context.Background(), planFile)
	if err != nil {
		Error("Failed to parse plan: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return nil, err
	}

	return show, nil
}

func summarizeChanges(show *tfjson.Plan) {
	added, changed, destroyed := 0, 0, 0

	for _, resource := range show.ResourceChanges {
		for _, action := range resource.Change.Actions {
			switch action {
			case tfjson.ActionCreate:
				added++
			case tfjson.ActionUpdate:
				changed++
			case tfjson.ActionDelete:
				destroyed++
			}
		}
	}

	Success("Apply complete! Resources: %d added, %d changed, %d destroyed", added, changed, destroyed)
}

func askForApproval() bool {
	var input string
	fmt.Print("\nDo you want to perform these actions? Only 'yes' will be accepted to approve.\nEnter a value: ")
	fmt.Scanln(&input)
	fmt.Println()
	return input == "yes"
}

func colorizeNoChanges(plan string) string {
	lines := strings.Split(plan, "\n")

	for i, line := range lines {
		if strings.Contains(line, "No changes.") ||
			strings.Contains(line, "Your infrastructure matches the configuration") ||
			strings.Contains(line, "no differences") ||
			strings.Contains(line, "no changes are needed") {
			lines[i] = fmt.Sprintf("\033[32m%s\033[0m", line)
		}
	}

	return strings.Join(lines, "\n")
}

func cleanupPlanFile() {
	if _, err := os.Stat("plan.out"); err == nil {
		os.Remove("plan.out")
	}
}

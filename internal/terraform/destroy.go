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
func Destroy(approve bool, lock bool, dir string, vars []string, varFiles []string) error { // UPDATED: added new parameters
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform client: %v", err)
		return err
	}

	Info("Preparing Terraform destroy operation in directory: %s", dir)

	// Build plan options
	planOptions := []tfexec.PlanOption{
		tfexec.Destroy(true),
		tfexec.Out("plan.out"),
	}

	if len(vars) > 0 {
		Info("Setting %d variable(s)...", len(vars))
		for _, v := range vars {
			Info("Using variable: %s", v)
			planOptions = append(planOptions, tfexec.Var(v))
		}
	}

	if len(varFiles) > 0 {
		Info("Loading %d variable file(s)...", len(varFiles))
		for _, vf := range varFiles {
			if _, err := os.Stat(vf); os.IsNotExist(err) {
				Error("Variable file not found: %s", vf)
				return fmt.Errorf("variable file not found: %s", vf)
			}
			Info("Using var-file: %s", vf)
			planOptions = append(planOptions, tfexec.VarFile(vf))
		}
	}

	// Generate destroy plan
	_, err = tf.Plan(context.Background(), planOptions...)
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

	// Build apply options for destroy
	applyOptions := []tfexec.ApplyOption{
		tfexec.Destroy(true),
		tfexec.DirOrPlan("plan.out"),
		tfexec.Lock(lock),
	}

	err = tf.Apply(context.Background(), applyOptions...)
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

// DestroyLogger remains the same...
type DestroyLogger struct {
	CustomColorWriter
	isDestroy bool
}

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

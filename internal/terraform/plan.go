package terraform

import (
	"bytes"
	"context"
	"os"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pterm/pterm"
)

// Plan runs 'terraform plan' and outputs the plan to the console.
// It allows setting variables either via command-line arguments or variable files.
// The function provides user feedback through spinners and colored messages,
// and handles any errors that occur during the planning process.
func Plan(vars []string, varFiles []string, dir string, destroy bool) error {
	tf, err := GetTerraform(dir)
	if err != nil {
		return err
	}

	var outputBuffer bytes.Buffer
	customWriter := &CustomColorWriter{
		Buffer: &outputBuffer,
		Writer: os.Stdout,
	}

	tf.SetStdout(customWriter)
	tf.SetStderr(os.Stderr)

	pterm.Info.Println("Infrastucture planning...")
	spinner, _ := pterm.DefaultSpinner.Start("Infrastructure planning...")

	planOptions := []tfexec.PlanOption{}

	if vars != nil {
		pterm.Info.Printf("Setting variable: %s\n", vars)
		for _, v := range vars {
			planOptions = append(planOptions, tfexec.Var(v))
		}
	}

	if varFiles != nil {
		pterm.Info.Printf("Setting variable file: %s\n", varFiles)
		for _, vf := range varFiles {
			planOptions = append(planOptions, tfexec.VarFile(vf))
		}
	}

	// Add destroy flag if set
	if destroy {
		pterm.Info.Println("Planning for destruction")
		planOptions = append(planOptions, tfexec.Destroy(true))
	}

	_, err = tf.Plan(context.Background(), planOptions...)
	if err != nil {
		spinner.Fail("Plan failed")
		pterm.Error.Printf("Plan failed: %v\n", err)
		return err
	}

	spinner.Success("Infrastructure planning completed successfully")
	return nil
}

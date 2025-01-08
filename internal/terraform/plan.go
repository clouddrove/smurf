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
func Plan(varNameValue string, varFile string) error {
	tf, err := getTerraform()
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

	pterm.Info.Println("Infrastucture planing...")
	spinner, _ := pterm.DefaultSpinner.Start("Infrastructure planing...")

	if varNameValue != "" {
		pterm.Info.Printf("Setting variable: %s\n", varNameValue)
		tf.Plan(context.Background(), tfexec.Var(varNameValue))
	}

	if varFile != "" {
		pterm.Info.Printf("Setting variable file: %s\n", varFile)
		_, err = tf.Plan(context.Background(), tfexec.VarFile(varFile))
		if err != nil {
			spinner.Fail("Plan failed")
			pterm.Error.Printf("Plan failed: %v\n", err)
			return err
		}
	}

	_, err = tf.Plan(context.Background())
	if err != nil {
		spinner.Fail("Plan failed")
		pterm.Error.Printf("Plan failed: %v\n", err)
		return err
	}
	spinner.Success("Infrastructure planing completed successfully")

	return nil
}

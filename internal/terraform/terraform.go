package terraform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/clouddrove/smurf/configs"
	"github.com/fatih/color"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pterm/pterm"
)

// getTerraform locates the Terraform binary and initializes a Terraform instance
func getTerraform() (*tfexec.Terraform, error) {
	terraformBinary, err := exec.LookPath("terraform")
	if err != nil {
		pterm.Error.Println("Terraform binary not found in PATH. Please install Terraform.")
		return nil, err
	}

	tf, err := tfexec.NewTerraform(".", terraformBinary)
	if err != nil {
		pterm.Error.Printf("Error creating Terraform instance: %v\n", err)
		return nil, err
	}

	pterm.Success.Printf("Using Terraform binary at: %s\n", terraformBinary)
	return tf, nil
}

// Init initializes Terraform
func Init() error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	pterm.Info.Println("Initializing Terraform...")
	spinner, _ := pterm.DefaultSpinner.Start("Running terraform init")
	err = tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		spinner.Fail("Terraform init failed")
		pterm.Error.Printf("Terraform init failed: %v\n", err)
		return err
	}

	customWriter := &configs.CustomColorWriter{Writer: os.Stdout}

	tf.SetStdout(customWriter)
	tf.SetStderr(os.Stderr)

	spinner.Success("Terraform initialized successfully")

	pterm.Success.Println("Terraform configuration validated successfully.")
	return nil
}

// Validate checks the validity of the Terraform configuration
func Validate() error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	pterm.Info.Println("Validating Terraform configuration...")
	spinner, _ := pterm.DefaultSpinner.Start("Running terraform validate")

	valid, err := tf.Validate(context.Background())
	if err != nil {
		spinner.Fail("Terraform validation failed")
		pterm.Error.Printf("Terraform validation failed: %v\n", err)
		return err
	}

	customWriter := &configs.CustomColorWriter{Writer: os.Stdout}

	tf.SetStdout(customWriter)
	tf.SetStderr(os.Stderr)

	if valid.Valid {
		spinner.Success("Terraform configuration is valid.")
	} else {
		spinner.Fail("Terraform configuration is invalid.")
	}

	return nil
}

// Plan runs 'terraform plan' and outputs the plan to the console
func Plan(varNameValue string, varFile string) error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	var outputBuffer bytes.Buffer

	customWriter := &configs.CustomColorWriter{
		Buffer: &outputBuffer,
		Writer: os.Stdout,
	}

	tf.SetStdout(customWriter)
	tf.SetStderr(os.Stderr)

	pterm.Info.Println("Running Terraform plan...")
	spinner, _ := pterm.DefaultSpinner.Start("Running terraform plan")

	if varNameValue != "" {
		pterm.Info.Printf("Setting variable: %s\n", varNameValue)
		tf.Plan(context.Background(), tfexec.Var(varNameValue))
	}

	if varFile != "" {
		pterm.Info.Printf("Setting variable file: %s\n", varFile)
		_, err = tf.Plan(context.Background(), tfexec.VarFile(varFile))
		if err != nil {
			spinner.Fail("Terraform plan failed")
			pterm.Error.Printf("Terraform plan failed: %v\n", err)
			return err
		}
	}

	_, err = tf.Plan(context.Background())
	if err != nil {
		spinner.Fail("Terraform plan failed")
		pterm.Error.Printf("Terraform plan failed: %v\n", err)
		return err
	}
	spinner.Success("Terraform plan completed successfully")

	return nil
}

// Apply executes 'terraform apply' to apply the planned changes
func Apply() error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	pterm.Info.Println("Applying Terraform changes...")
	spinner, _ := pterm.DefaultSpinner.Start("Running terraform apply")
	err = tf.Apply(context.Background())
	if err != nil {
		spinner.Fail("Terraform apply failed")
		pterm.Error.Printf("Terraform apply failed: %v\n", err)
		return err
	}
	
	customWriter := &configs.CustomColorWriter{Writer: os.Stdout}

	tf.SetStdout(customWriter)
	tf.SetStderr(os.Stderr)
	
	spinner.Success("Terraform applied successfully.")

	return nil
}

// Destroy removes all resources managed by Terraform
func Destroy() error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	customWriter := &configs.CustomColorWriter{Writer: os.Stdout}

	tf.SetStdout(customWriter)
	tf.SetStderr(os.Stderr)

	pterm.Info.Println("Destroying Terraform resources...")
	spinner, _ := pterm.DefaultSpinner.Start("Running terraform destroy")
	err = tf.Destroy(context.Background())
	if err != nil {
		spinner.Fail("Terraform destroy failed")
		pterm.Error.Printf("Terraform destroy failed: %v\n", err)
		return err
	}
	spinner.Success("Terraform resources destroyed successfully.")

	return nil
}

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

// Output displays the outputs defined in the Terraform configuration
func Output() error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	tf.SetStdout(os.Stdout)
	tf.SetStderr(os.Stderr)

	pterm.Info.Println("Refreshing Terraform state...")
	spinner, _ := pterm.DefaultSpinner.Start("Running terraform refresh")
	err = tf.Refresh(context.Background())
	if err != nil {
		spinner.Fail("Error refreshing Terraform state")
		pterm.Error.Printf("Error refreshing Terraform state: %v\n", err)
		return err
	}
	spinner.Success("Terraform state refreshed successfully.")

	outputs, err := tf.Output(context.Background())
	if err != nil {
		pterm.Error.Printf("Error getting Terraform outputs: %v\n", err)
		return err
	}

	if len(outputs) == 0 {
		pterm.Info.Println("No outputs found.")
		return nil
	}

	green := color.New(color.FgGreen).SprintfFunc()

	pterm.Info.Println("Terraform outputs:")
	for key, value := range outputs {
		if value.Sensitive {
			fmt.Println(green("%s: [sensitive value hidden]", key))
		} else {
			fmt.Println(green("%s: %v", key, value.Value))
		}
	}

	return nil
}

// Format applies a canonical format to Terraform configuration files
func Format() error {
	tf, err := getTerraform()
	if err != nil {
		return err
	}

	pterm.Info.Println("Formatting Terraform configuration files...")
	spinner, _ := pterm.DefaultSpinner.Start("Running terraform fmt")

	cmd := exec.Command(tf.ExecPath(), "fmt")

	cmd.Dir = "." 

	output, err := cmd.CombinedOutput()
	if err != nil {
		spinner.Fail("Terraform format failed")
		pterm.Error.Printf("Terraform format failed: %v\nOutput: %s\n", err, string(output))
		return err
	}
	spinner.Success("Terraform configuration files formatted successfully")

	return nil
}

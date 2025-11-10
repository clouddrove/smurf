package terraform

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

// Refresh updates the state file of your infrastructure.
func Refresh(vars []string, varFiles []string, lock bool, dir string) error {
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform: %v", err)
		return err
	}

	Info("Refreshing Terraform state...")
	applyOptions := []tfexec.RefreshCmdOption{}

	// Handle variables
	if vars != nil {
		for _, v := range vars {
			Info("Using variable: %s", v)
			applyOptions = append(applyOptions, tfexec.Var(v))
		}
	}

	// Handle variable files
	if varFiles != nil {
		for _, vf := range varFiles {
			Info("Using variable file: %s", vf)
			applyOptions = append(applyOptions, tfexec.VarFile(vf))
		}
	}

	// Warn if locking is disabled
	if !lock {
		Warn("State locking is disabled! This may cause conflicts in concurrent operations.")
	}

	startTime := time.Now()

	// Show existing state before refresh
	currentState, err := tf.Show(context.Background())
	if err == nil && currentState != nil && currentState.Values != nil {
		resources := getAllRefreshResources(currentState.Values.RootModule)
		for _, resource := range resources {
			idVal, _ := resource.AttributeValues["id"].(string)
			Info("%s: Refreshing state... [id=%s]", resource.Address, idVal)
		}
	}

	// Execute refresh
	err = tf.Refresh(context.Background(), applyOptions...)
	if err != nil {
		Error("Terraform refresh failed: %v", err)
		Error("The above error occurred while Terraform attempted to refresh the state file.")
		return err
	}

	// Display updated state info
	state, err := tf.Show(context.Background())
	if err != nil {
		Error("Error reading updated state: %v", err)
		return fmt.Errorf("error reading updated state: %v", err)
	}

	// Log completion
	Success("Refreshed %d resources successfully.", countResources(state.Values.RootModule))

	duration := time.Since(startTime).Round(time.Second)
	Info("Execution time: %s", duration)

	return nil
}

// getAllRefreshResources recursively collects all resources from all modules.
func getAllRefreshResources(module *tfjson.StateModule) []*tfjson.StateResource {
	if module == nil {
		return nil
	}

	resources := module.Resources
	for _, childModule := range module.ChildModules {
		resources = append(resources, getAllRefreshResources(childModule)...)
	}
	return resources
}

// countResources recursively counts resources in all modules.
func countResources(module *tfjson.StateModule) int {
	if module == nil {
		return 0
	}

	count := len(module.Resources)
	for _, childModule := range module.ChildModules {
		count += countResources(childModule)
	}
	return count
}

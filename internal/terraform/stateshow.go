package terraform

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/clouddrove/smurf/internal/ai"
	tfjson "github.com/hashicorp/terraform-json"
)

// StateShow shows details of a specific resource in the Terraform state
func StateShow(dir string, address string, useAI bool) error {
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	state, err := tf.Show(context.Background())
	if err != nil {
		Error("Unable to read Terraform state: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return fmt.Errorf("failed to read state: %v", err)
	}

	if state == nil || state.Values == nil || state.Values.RootModule == nil {
		Warn("No resources found in the current Terraform state.")
		return nil
	}

	// Find the specific resource
	resource := findResourceByAddress(state.Values.RootModule, address)
	if resource == nil {
		Error("Resource not found: %s", address)
		return fmt.Errorf("resource not found: %s", address)
	}

	// Output the resource details as JSON
	output, err := json.MarshalIndent(resource, "", "  ")
	if err != nil {
		Error("Failed to format resource output: %v", err)
		return err
	}

	fmt.Println(string(output))
	return nil
}

// findResourceByAddress recursively searches for a resource by its address
func findResourceByAddress(module *tfjson.StateModule, address string) *tfjson.StateResource {
	if module == nil {
		return nil
	}

	// Check resources in current module
	for _, resource := range module.Resources {
		if resource.Address == address {
			return resource
		}
	}

	// Check child modules
	for _, childModule := range module.ChildModules {
		if resource := findResourceByAddress(childModule, address); resource != nil {
			return resource
		}
	}

	return nil
}

package terraform

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/clouddrove/smurf/internal/ai"
	tfjson "github.com/hashicorp/terraform-json"
)

// StateList lists all Terraform resources currently tracked in the state file.
func StateList(dir string, useAI bool) error {
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

	// No resources found
	if state == nil || state.Values == nil || state.Values.RootModule == nil {
		Warn("No resources found in the current Terraform state.")
		return nil
	}

	// Collect all resource addresses
	resources := getAllResources(state.Values.RootModule)
	sort.Strings(resources)

	Info("Resources found in Terraform state:")
	for _, addr := range resources {
		fmt.Printf("  %s\n", addr)
	}

	Success("Total %d resources listed.", len(resources))
	return nil
}

// getAllResources recursively collects resources from all modules
func getAllResources(module *tfjson.StateModule) []string {
	if module == nil {
		return nil
	}

	var addresses []string
	for _, resource := range module.Resources {
		// Skip data sources, only show managed resources
		if !strings.HasPrefix(resource.Type, "data.") {
			addresses = append(addresses, resource.Address)
		}
	}

	for _, childModule := range module.ChildModules {
		addresses = append(addresses, getAllResources(childModule)...)
	}

	return addresses
}

// ErrorHandler provides consistent CLI error output across Terraform operations.
func ErrorHandler(err error) {
	if err != nil {
		Error("Command failed: %v", err)
	}
}

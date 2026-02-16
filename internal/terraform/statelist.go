package terraform

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/clouddrove/smurf/internal/ai"
	tfjson "github.com/hashicorp/terraform-json"
)

// StateList lists resources in the Terraform state using tf.Show()
func StateList(dir string, addresses []string, useAI bool) error {
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	Info("Listing resources in Terraform state from directory: %s", dir)

	state, err := tf.Show(context.Background())
	if err != nil {
		Error("Failed to read Terraform state: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	if state == nil || state.Values == nil || state.Values.RootModule == nil {
		Warn("No resources found in state")
		return nil
	}

	resources := getAllResources(state.Values.RootModule)

	if len(addresses) > 0 {
		var filtered []string
		for _, addr := range resources {
			for _, filter := range addresses {
				if strings.Contains(addr, filter) {
					filtered = append(filtered, addr)
					break
				}
			}
		}
		resources = filtered
	}

	sort.Strings(resources)

	if len(resources) == 0 {
		Warn("No matching resources found")
		return nil
	}

	Success("Found %d resource(s):", len(resources))
	for _, r := range resources {
		fmt.Println(r)
	}

	return nil
}

func getAllResources(module *tfjson.StateModule) []string {
	if module == nil {
		return nil
	}
	var resources []string
	for _, r := range module.Resources {
		if !strings.HasPrefix(r.Type, "data.") {
			resources = append(resources, r.Address)
		}
	}
	for _, child := range module.ChildModules {
		resources = append(resources, getAllResources(child)...)
	}
	return resources
}

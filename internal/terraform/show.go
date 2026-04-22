package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/clouddrove/smurf/internal/ai"
	tfjson "github.com/hashicorp/terraform-json"
)

// ShowState displays the current Terraform state
func ShowState(vars []string, varFiles []string, dir string, jsonOutput bool, useAI bool) error {
	Step("Initializing Terraform client...")
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform client: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	Step("Fetching Terraform state...")

	// Variables don't apply to show state - warn user if they provide them
	if len(vars) > 0 {
		Warn("Note: -var flags are ignored when showing state (variables don't apply to state display)")
	}
	if len(varFiles) > 0 {
		Warn("Note: -var-file flags are ignored when showing state (variables don't apply to state display)")
	}

	if jsonOutput {
		// Use Show for JSON output
		state, err := tf.Show(context.Background())
		if err != nil {
			Error("Failed to show state: %v", err)
			ai.AIExplainError(useAI, err.Error())
			return err
		}
		// Convert state to JSON string
		jsonOutput, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			Error("Failed to marshal state to JSON: %v", err)
			return err
		}
		fmt.Println(string(jsonOutput))
	} else {
		// For human-readable output, get the state as JSON
		state, err := tf.Show(context.Background())
		if err != nil {
			Error("Failed to show state: %v", err)
			ai.AIExplainError(useAI, err.Error())
			return err
		}

		// Pretty print the state
		printStateHumanReadable(state)
	}

	Success("State displayed successfully")
	return nil
}

// ShowResource displays a specific resource from the state
func ShowResource(resourceAddr string, vars []string, varFiles []string, dir string, jsonOutput bool, useAI bool) error {
	Step("Initializing Terraform client...")
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform client: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	Step("Showing resource: %s", resourceAddr)

	// Variables don't apply to show resource - warn user if they provide them
	if len(vars) > 0 {
		Warn("Note: -var flags are ignored when showing resource (variables don't apply to state display)")
	}
	if len(varFiles) > 0 {
		Warn("Note: -var-file flags are ignored when showing resource (variables don't apply to state display)")
	}

	// Get the full state
	state, err := tf.Show(context.Background())
	if err != nil {
		Error("Failed to get state: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	// Find the specific resource
	var foundResource *tfjson.StateResource
	if state.Values != nil && state.Values.RootModule != nil {
		// Search in root module
		for _, resource := range state.Values.RootModule.Resources {
			if resource.Address == resourceAddr {
				foundResource = resource
				break
			}
		}

		// Search in child modules if not found
		if foundResource == nil && state.Values.RootModule.ChildModules != nil {
			foundResource = searchResourceInModules(state.Values.RootModule.ChildModules, resourceAddr)
		}
	}

	if foundResource == nil {
		Error("Resource not found: %s", resourceAddr)
		return fmt.Errorf("resource not found: %s", resourceAddr)
	}

	if jsonOutput {
		output, err := json.MarshalIndent(foundResource, "", "  ")
		if err != nil {
			Error("Failed to marshal resource to JSON: %v", err)
			return err
		}
		fmt.Println(string(output))
	} else {
		printResourceHumanReadable(foundResource)
	}

	Success("Resource details displayed successfully")
	return nil
}

// ShowPlan displays details of a saved plan file
func ShowPlan(planFile string, vars []string, varFiles []string, dir string, jsonOutput bool, useAI bool) error {
	Step("Initializing Terraform client...")
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform client: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	// Check if plan file exists
	if _, err := os.Stat(planFile); os.IsNotExist(err) {
		Error("Plan file not found: %s", planFile)
		ai.AIExplainError(useAI, fmt.Sprintf("Plan file not found: %s", planFile))
		return fmt.Errorf("plan file not found: %s", planFile)
	}

	Step("Showing plan from file: %s", planFile)

	// Warn users that vars don't apply to saved plans
	if len(vars) > 0 {
		Warn("Note: -var flags are ignored when showing an existing plan file (variables are already saved in the plan)")
	}
	if len(varFiles) > 0 {
		Warn("Note: -var-file flags are ignored when showing an existing plan file (variables are already saved in the plan)")
	}

	if jsonOutput {
		// Get raw JSON output
		output, err := tf.ShowPlanFileRaw(context.Background(), planFile)
		if err != nil {
			Error("Failed to read plan: %v", err)
			ai.AIExplainError(useAI, err.Error())
			return err
		}
		fmt.Println(string(output))
	} else {
		// Get raw output for human-readable display with colorization
		output, err := tf.ShowPlanFileRaw(context.Background(), planFile)
		if err != nil {
			Error("Failed to read plan: %v", err)
			ai.AIExplainError(useAI, err.Error())
			return err
		}

		// Colorize and display the plan
		coloredOutput := colorizePlanOutput(string(output))
		fmt.Print(coloredOutput)

		// Also show summary from parsed plan
		plan, err := tf.ShowPlanFile(context.Background(), planFile)
		if err == nil && plan != nil {
			summarizePlanChanges(plan)
		}
	}

	Success("Plan displayed successfully")
	return nil
}

// Helper function to search for resource in child modules recursively
func searchResourceInModules(modules []*tfjson.StateModule, resourceAddr string) *tfjson.StateResource {
	for _, module := range modules {
		// Check resources in current module
		for _, resource := range module.Resources {
			if resource.Address == resourceAddr {
				return resource
			}
		}

		// Recursively search in child modules
		if len(module.ChildModules) > 0 {
			if found := searchResourceInModules(module.ChildModules, resourceAddr); found != nil {
				return found
			}
		}
	}
	return nil
}

// Helper function to print state in human-readable format
func printStateHumanReadable(state *tfjson.State) {
	if state == nil || state.Values == nil {
		fmt.Println("No state found")
		return
	}

	fmt.Println("\n\033[1mResources:\033[0m")
	if state.Values.RootModule != nil {
		printResourcesFromModule(state.Values.RootModule, 0)
	}

	// Print outputs if any
	if state.Values.Outputs != nil && len(state.Values.Outputs) > 0 {
		fmt.Println("\n\033[1mOutputs:\033[0m")
		for name, output := range state.Values.Outputs {
			fmt.Printf("  \033[33m%s\033[0m = %v\n", name, output.Value)
		}
	}
}

// Helper function to print resources from a module recursively
func printResourcesFromModule(module *tfjson.StateModule, indent int) {
	indentStr := strings.Repeat("  ", indent)

	for _, resource := range module.Resources {
		fmt.Printf("%s\033[36m%s\033[0m\n", indentStr, resource.Address)
		fmt.Printf("%s  Type: %s\n", indentStr, resource.Type)
		fmt.Printf("%s  Provider: %s\n", indentStr, resource.ProviderName)

		// Print important attributes
		if id, ok := resource.AttributeValues["id"]; ok {
			fmt.Printf("%s  ID: %v\n", indentStr, id)
		}
		if arn, ok := resource.AttributeValues["arn"]; ok {
			fmt.Printf("%s  ARN: %v\n", indentStr, arn)
		}
		fmt.Println()
	}

	// Print resources from child modules
	for _, childModule := range module.ChildModules {
		printResourcesFromModule(childModule, indent+1)
	}
}

// Helper function to print resource in human-readable format
func printResourceHumanReadable(resource *tfjson.StateResource) {
	fmt.Printf("\n\033[1mResource: %s\033[0m\n", resource.Address)
	fmt.Printf("  Type: %s\n", resource.Type)
	fmt.Printf("  Provider: %s\n", resource.ProviderName)

	fmt.Println("\n  \033[1mAttributes:\033[0m")
	for key, value := range resource.AttributeValues {
		// Skip sensitive values or very long values for readability
		valueStr := fmt.Sprintf("%v", value)
		if len(valueStr) > 100 {
			valueStr = valueStr[:97] + "..."
		}
		fmt.Printf("    \033[36m%s\033[0m = %s\n", key, valueStr)
	}
}

// Helper function to colorize plan output
func colorizePlanOutput(output string) string {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		switch {
		case strings.Contains(line, "Plan:"):
			lines[i] = fmt.Sprintf("\033[1m%s\033[0m", line)
		case strings.Contains(line, "to add"):
			lines[i] = fmt.Sprintf("\033[32m%s\033[0m", line)
		case strings.Contains(line, "to change"):
			lines[i] = fmt.Sprintf("\033[33m%s\033[0m", line)
		case strings.Contains(line, "to destroy"):
			lines[i] = fmt.Sprintf("\033[31m%s\033[0m", line)
		case strings.Contains(line, "No changes."):
			lines[i] = fmt.Sprintf("\033[36m%s\033[0m", line)
		case strings.Contains(line, "Terraform will perform the following actions:"):
			lines[i] = fmt.Sprintf("\033[1m%s\033[0m", line)
		case strings.Contains(line, "~"):
			lines[i] = fmt.Sprintf("\033[33m%s\033[0m", line)
		case strings.Contains(line, "+"):
			lines[i] = fmt.Sprintf("\033[32m%s\033[0m", line)
		case strings.Contains(line, "-"):
			lines[i] = fmt.Sprintf("\033[31m%s\033[0m", line)
		}
	}
	return strings.Join(lines, "\n")
}

// Helper function to summarize plan changes
func summarizePlanChanges(plan *tfjson.Plan) {
	if plan == nil || plan.ResourceChanges == nil {
		return
	}

	added, changed, destroyed := 0, 0, 0

	for _, resource := range plan.ResourceChanges {
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

	if added > 0 || changed > 0 || destroyed > 0 {
		fmt.Printf("\n\033[1mSummary:\033[0m \033[32m%d to add\033[0m, \033[33m%d to change\033[0m, \033[31m%d to destroy\033[0m\n", added, changed, destroyed)
	}
}

package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/clouddrove/smurf/internal/ai"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/pterm/pterm"
)

// ShowState displays the current Terraform state
func ShowState(vars []string, varFiles []string, dir string, jsonOutput bool, useAI bool) error {
	Step("Initializing Terraform client for Show State...")
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
		stateJSON, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			Error("Failed to marshal state to JSON: %v", err)
			return err
		}
		fmt.Println(string(stateJSON))
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
	Step("Initializing Terraform client for Show Resource...")
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform client for shoe resource: %v", err)
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
	Step("Initializing Terraform client for Show Plan...")
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform client for show plan: %v", err)
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
		// Parse the plan for structured output
		plan, err := tf.ShowPlanFile(context.Background(), planFile)
		if err != nil {
			Error("Failed to read plan: %v", err)
			ai.AIExplainError(useAI, err.Error())
			return err
		}

		if plan == nil {
			Error("Plan is empty or invalid")
			return fmt.Errorf("plan is empty or invalid")
		}

		// Print human-readable plan summary
		printPlanHumanReadable(plan)
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
		pterm.Info.Println("No state found")
		return
	}

	pterm.DefaultSection.Println("Resources")
	if state.Values.RootModule != nil {
		printResourcesFromModule(state.Values.RootModule, 0)
	}

	// Print outputs if any
	if state.Values.Outputs != nil && len(state.Values.Outputs) > 0 {
		pterm.DefaultSection.Println("Outputs")
		for name, output := range state.Values.Outputs {
			pterm.Printf("  %s = %v\n", pterm.FgYellow.Sprint(name), output.Value)
		}
	}
}

// Helper function to print resources from a module recursively
func printResourcesFromModule(module *tfjson.StateModule, indent int) {
	indentStr := strings.Repeat("  ", indent)

	for _, resource := range module.Resources {
		pterm.Printf("%s%s\n", indentStr, pterm.FgCyan.Sprint(resource.Address))
		pterm.Printf("%s  Type: %s\n", indentStr, resource.Type)
		pterm.Printf("%s  Provider: %s\n", indentStr, resource.ProviderName)

		// Print important attributes
		if id, ok := resource.AttributeValues["id"]; ok {
			pterm.Printf("%s  ID: %v\n", indentStr, id)
		}
		if arn, ok := resource.AttributeValues["arn"]; ok {
			pterm.Printf("%s  ARN: %v\n", indentStr, arn)
		}
		pterm.Println()
	}

	// Print resources from child modules
	for _, childModule := range module.ChildModules {
		printResourcesFromModule(childModule, indent+1)
	}
}

// Helper function to print resource in human-readable format
func printResourceHumanReadable(resource *tfjson.StateResource) {
	pterm.DefaultSection.Println(pterm.Bold.Sprintf("Resource: %s", resource.Address))
	pterm.Printf("  Type: %s\n", resource.Type)
	pterm.Printf("  Provider: %s\n", resource.ProviderName)

	pterm.Println()
	pterm.DefaultSection.Println("Attributes")
	for key, value := range resource.AttributeValues {
		// Skip sensitive values or very long values for readability
		valueStr := fmt.Sprintf("%v", value)
		if len(valueStr) > 100 {
			valueStr = valueStr[:97] + "..."
		}
		pterm.Printf("  %s = %s\n", pterm.FgCyan.Sprint(key), valueStr)
	}
}

// Helper function to print plan in human-readable format with proper styling
func printPlanHumanReadable(plan *tfjson.Plan) {
	if plan == nil {
		pterm.Info.Println("No plan found")
		return
	}

	// Print resource changes
	if plan.ResourceChanges != nil && len(plan.ResourceChanges) > 0 {
		pterm.DefaultSection.Println("Resource Changes")

		for _, rc := range plan.ResourceChanges {
			// Print resource address with appropriate color based on action
			var addressPrefix string
			var addressColor pterm.Color
			switch {
			case containsAction(rc.Change.Actions, tfjson.ActionCreate):
				addressPrefix = "+"
				addressColor = pterm.FgGreen
			case containsAction(rc.Change.Actions, tfjson.ActionDelete):
				addressPrefix = "-"
				addressColor = pterm.FgRed
			default:
				addressPrefix = "~"
				addressColor = pterm.FgYellow
			}

			pterm.Printf("  %s %s\n",
				addressColor.Sprint(addressPrefix),
				addressColor.Sprint(rc.Address))
			pterm.Printf("    Type: %s\n", rc.Type)
			pterm.Printf("    Provider: %s\n", rc.ProviderName)

			// Show action summary
			actionStr := formatActions(rc.Change.Actions)
			pterm.Printf("    Actions: %s\n", actionStr)
			pterm.Println()
		}
	}

	// Print summary
	summarizePlanChanges(plan)
}

// Helper function to check if actions slice contains a specific action
func containsAction(actions []tfjson.Action, action tfjson.Action) bool {
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}

// Helper function to format actions for display
func formatActions(actions []tfjson.Action) string {
	var formatted []string
	for _, action := range actions {
		switch action {
		case tfjson.ActionCreate:
			formatted = append(formatted, pterm.FgGreen.Sprint("create"))
		case tfjson.ActionUpdate:
			formatted = append(formatted, pterm.FgYellow.Sprint("update"))
		case tfjson.ActionDelete:
			formatted = append(formatted, pterm.FgRed.Sprint("delete"))
		case tfjson.ActionRead:
			formatted = append(formatted, pterm.FgCyan.Sprint("read"))
		default:
			formatted = append(formatted, string(action))
		}
	}
	return strings.Join(formatted, ", ")
}

// Helper function to summarize plan changes
func summarizePlanChanges(plan *tfjson.Plan) {
	if plan == nil || plan.ResourceChanges == nil {
		return
	}

	added, changed, destroyed := 0, 0, 0

	for _, resource := range plan.ResourceChanges {
		if containsAction(resource.Change.Actions, tfjson.ActionCreate) {
			added++
		}
		if containsAction(resource.Change.Actions, tfjson.ActionUpdate) {
			changed++
		}
		if containsAction(resource.Change.Actions, tfjson.ActionDelete) {
			destroyed++
		}
	}

	if added > 0 || changed > 0 || destroyed > 0 {
		summaryText := pterm.Sprintf("%s, %s, %s",
			pterm.FgGreen.Sprintf("%d to add", added),
			pterm.FgYellow.Sprintf("%d to change", changed),
			pterm.FgRed.Sprintf("%d to destroy", destroyed))
		pterm.DefaultSection.Println("Summary")
		pterm.Println(summaryText)
	}
}

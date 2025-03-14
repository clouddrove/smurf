package terraform

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/pterm/pterm"
)

// StateList displays all resources in the Terraform state
func StateList(dir string) error {
	tf, err := GetTerraform(dir)
	spinner, err := pterm.DefaultSpinner.Start("Reading state...")
	if err != nil {
		return err
	}

	state, err := tf.Show(context.Background())
	if err != nil {
		spinner.Fail("Failed to read state")
		return fmt.Errorf("failed to read state: %v", err)
	}

	if state == nil || state.Values == nil || state.Values.RootModule == nil {
		fmt.Print(color.GreenString("INFO "))
		spinner.Success("State read successfully")
		fmt.Println("No resources found in state.")
		return nil
	}

	resources := getAllResources(state.Values.RootModule)

	sort.Strings(resources)

	for _, addr := range resources {
		fmt.Println(addr)
	}

	spinner.Success("State read successfully")

	return nil
}

// getAllResources recursively collects resources from all modules
func getAllResources(module *tfjson.StateModule) []string {
	if module == nil {
		return nil
	}

	var addresses []string

	for _, resource := range module.Resources {
		if !strings.HasPrefix(resource.Type, "data.") {
			addresses = append(addresses, resource.Address)
		}
	}

	for _, childModule := range module.ChildModules {
		addresses = append(addresses, getAllResources(childModule)...)
	}

	return addresses
}

// ErrorHandler handles CLI errors
func ErrorHandler(err error) {
	if err != nil {
		errMsg := color.RedString("Error: ")
		fmt.Printf("%s%v\n", errMsg, err)
	}
}

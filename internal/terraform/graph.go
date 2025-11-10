package terraform

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

// Graph generates a visual representation of Terraform resources.
// It produces the Terraform DOT graph output, which can be visualized with Graphviz.
// Uses Smurf unified logs for consistent and readable output.
func Graph(dir string) error {
	tf, err := GetTerraform(dir)
	if err != nil {
		Error("Failed to initialize Terraform: %v", err)
		return err
	}

	Info("Starting Terraform graph generation...")

	graphDOT, err := tf.Graph(context.Background(), tfexec.DrawCycles(true))
	if err != nil {
		Error("Error generating Terraform graph: %v", err)
		return err
	}

	fmt.Println()
	Success("Terraform Dependency Graph (DOT Format):")
	fmt.Println("-------------------------------------------------")
	fmt.Println(graphDOT)
	fmt.Println("-------------------------------------------------")
	Info("You can visualize this graph using Graphviz, e.g.:\n  dot -Tpng graph.dot -o graph.png")

	return nil
}

// generateDOTGraph creates a DOT graph from the Terraform state.
// This is an alternate approach for rendering using tfjson.State.
func generateDOTGraph(state *tfjson.State) string {
	var sb strings.Builder

	sb.WriteString("digraph G {\n")
	sb.WriteString("  rankdir = \"RL\";\n")
	sb.WriteString("  node [shape = rect, fontname = \"sans-serif\"];\n\n")

	if state != nil && state.Values != nil && state.Values.RootModule != nil {
		processModule(&sb, state.Values.RootModule, "", make(map[string]bool))
	}

	sb.WriteString("}\n")
	return sb.String()
}

// processModule processes a Terraform module and its resources
// to generate a DOT-compatible graph structure.
func processModule(sb *strings.Builder, module *tfjson.StateModule, prefix string, processedNodes map[string]bool) {
	if module == nil {
		return
	}

	moduleName := strings.TrimPrefix(prefix, ".")
	if moduleName != "" {
		sb.WriteString(fmt.Sprintf("  subgraph \"cluster_%s\" {\n", moduleName))
		sb.WriteString(fmt.Sprintf("    label = \"%s\";\n", moduleName))
		sb.WriteString("    fontname = \"sans-serif\";\n")
	}

	for _, resource := range module.Resources {
		resourceAddr := getResourceAddress(resource, prefix)
		nodeName := fmt.Sprintf("\"%s\"", resourceAddr)
		label := strings.TrimPrefix(resource.Address, prefix+".")

		if !processedNodes[resourceAddr] {
			sb.WriteString(fmt.Sprintf("    %s [label=\"%s\"];\n", nodeName, label))
			processedNodes[resourceAddr] = true
		}
	}

	if moduleName != "" {
		sb.WriteString("  }\n\n")
	}

	for _, resource := range module.Resources {
		resourceAddr := getResourceAddress(resource, prefix)

		for _, dep := range resource.DependsOn {
			depAddr := getResourceAddress(&tfjson.StateResource{Address: dep}, prefix)
			sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\";\n", resourceAddr, depAddr))
		}
	}

	for _, child := range module.ChildModules {
		newPrefix := prefix
		if newPrefix == "" {
			newPrefix = child.Address
		} else {
			newPrefix = prefix + "." + strings.TrimPrefix(child.Address, prefix+".")
		}
		processModule(sb, child, newPrefix, processedNodes)
	}
}

// getResourceAddress returns the full resource address including prefixes.
func getResourceAddress(resource *tfjson.StateResource, prefix string) string {
	if prefix == "" {
		return resource.Address
	}
	return prefix + "." + strings.TrimPrefix(resource.Address, prefix+".")
}

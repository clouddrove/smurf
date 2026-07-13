package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/clouddrove/smurf/internal/utils"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pterm/pterm"
)

// Output displays the outputs defined in the Terraform configuration.
// It refreshes the Terraform state to ensure it reflects the current infrastructure,
// then retrieves and displays the outputs. Sensitive outputs are hidden for security.
//
// format selects the output shape: "table" (default) keeps the existing
// spinner-free but pterm-colored human output; "json" prints the outputs map
// as a single JSON document to stdout and suppresses every other stdout
// write (progress messages, the underlying `terraform` process output, AI
// explanations), so pipelines consuming stdout only ever see that document.
func Output(dir, format string, useAI bool) error {
	isTable := format == "" || format == "table"

	if !isTable {
		// GetTerraform prints via pterm on failure (e.g. "terraform binary
		// not found"); route that to stderr so stdout stays JSON-only, and
		// restore the default writer on return so no code path leaves the
		// global redirected.
		pterm.SetDefaultOutput(os.Stderr)
		defer pterm.SetDefaultOutput(os.Stdout)
	}

	tf, err := GetTerraform(dir)
	if err != nil {
		return err
	}

	if isTable {
		tf.SetStdout(os.Stdout)
		tf.SetStderr(os.Stderr)
	} else {
		// Keep the underlying `terraform` process' own stdout (e.g. the
		// refresh progress text) off real stdout so it can't mix with the
		// JSON document; discard it (outputs are read via tf.Output).
		tf.SetStdout(io.Discard)
		tf.SetStderr(os.Stderr)
	}

	if isTable {
		pterm.Info.Println("Refreshing Infrastructure state...")
	}
	if err := tf.Refresh(context.Background()); err != nil {
		if isTable {
			pterm.Error.Printf("Error refreshing  state: %v\n", err)
			ai.AIExplainError(useAI, err.Error())
		}
		return err
	}

	outputs, err := tf.Output(context.Background())
	if err != nil {
		if isTable {
			pterm.Error.Printf("Error getting Infrastructure outputs: %v\n", err)
			ai.AIExplainError(useAI, err.Error())
		}
		return err
	}

	if !isTable {
		return utils.PrintJSON(outputsToJSON(outputs))
	}

	if len(outputs) == 0 {
		pterm.Info.Println("No outputs found.")
		return nil
	}

	pterm.Info.Println("Outputs: ")
	for key, value := range outputs {
		if value.Sensitive {
			fmt.Println(pterm.Green("%s: [sensitive value hidden]", key))
		} else {
			fmt.Println(pterm.Green("%s: %v", key, value.Value))
		}
	}

	return nil
}

// outputsToJSON converts terraform's raw output map (whose values are
// themselves JSON-encoded) into a plain JSON-safe map, decoding each value
// and replacing sensitive ones with a placeholder rather than the real value.
func outputsToJSON(outputs map[string]tfexec.OutputMeta) map[string]interface{} {
	result := make(map[string]interface{}, len(outputs))
	for key, meta := range outputs {
		if meta.Sensitive {
			result[key] = "[sensitive value hidden]"
			continue
		}
		var v interface{}
		if err := json.Unmarshal(meta.Value, &v); err != nil {
			result[key] = string(meta.Value)
			continue
		}
		result[key] = v
	}
	return result
}

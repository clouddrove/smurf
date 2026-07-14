package docker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/distribution/reference"
	"github.com/pterm/pterm"
)

// Trivy runs 'trivy image' to scan a Docker image for vulnerabilities
// and displays the results. It's a simplified version that accepts just the image name and tag.
//
// format selects the output shape: "table" (default) keeps the existing
// pterm-wrapped human output; "json" asks trivy itself for JSON (via its own
// --format flag) and prints that document, and nothing else, to stdout, so
// pipelines consuming stdout only ever see it.
func Trivy(dockerImage, format string, useAI bool) error {
	isTable := format == "" || format == "table"

	if _, err := reference.ParseNormalizedNamed(dockerImage); err != nil {
		return fmt.Errorf("invalid image reference %q: %w", dockerImage, err)
	}

	ctx := context.Background()
	trivyFormat := "table"
	if !isTable {
		trivyFormat = "json"
	}
	args := []string{"image", dockerImage, "--format", trivyFormat}

	cmd := exec.CommandContext(ctx, "trivy", args...) //nolint:gosec // image ref validated above; args are not shell-expanded
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if isTable {
		pterm.Info.Println("Running 'trivy image' scan...")
	}

	err := cmd.Run()

	outStr := stdoutBuf.String()
	errStr := stderrBuf.String()

	if err != nil {
		if isTable {
			pterm.Error.Println("Error running 'trivy image':", err)
			if errStr != "" {
				pterm.Error.Println(errStr)
			}
			pterm.Error.Printfln("failed to run 'trivy image : %v", err)
			ai.AIExplainError(useAI, err.Error())
		}
		return fmt.Errorf("failed to run 'trivy image : %v", err)
	}

	if !isTable {
		fmt.Println(outStr)
		return nil
	}

	if outStr != "" {
		pterm.Info.Println("Trivy scan results : ", outStr)
	}

	pterm.Success.Println("Scan completed successfully.")
	return nil
}

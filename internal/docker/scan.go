package docker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/pterm/pterm"
)

// Trivy runs 'trivy image' to scan a Docker image for vulnerabilities
// and displays the results. It's a simplified version that accepts just the image name and tag.
func Trivy(dockerImage string, useAI bool) error {
	ctx := context.Background()
	args := []string{"image", dockerImage, "--format", "table"}

	cmd := exec.CommandContext(ctx, "trivy", args...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	pterm.Info.Println("Running 'trivy image' scan...")

	err := cmd.Run()

	outStr := stdoutBuf.String()
	errStr := stderrBuf.String()

	if err != nil {
		pterm.Error.Println("Error running 'trivy image':", err)
		if errStr != "" {
			pterm.Error.Println(errStr)
		}
		pterm.Error.Printfln("failed to run 'trivy image : %v", err)
		ai.AIExplainError(useAI, err.Error())
		return fmt.Errorf("failed to run 'trivy image : %v", err)
	}

	if outStr != "" {
		pterm.Info.Println("Trivy scan results : ", outStr)
	}

	pterm.Success.Println("Scan completed successfully.")
	return nil
}

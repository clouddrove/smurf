package docker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
)

// Trivy runs 'trivy image' to scan a Docker image for vulnerabilities
// and displays the results. It's a simplified version that accepts just the image name and tag.
func Trivy(dockerImage string) error {
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
		return logAndReturnError("failed to run 'trivy image : %v", err)
	}

	if outStr != "" {
		pterm.Info.Println("Trivy scan results:")
		fmt.Println(color.YellowString(outStr))
	}

	pterm.Success.Println("Scan completed successfully.")
	return nil
}

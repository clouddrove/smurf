package docker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Trivy runs 'trivy image' to scan a Docker image for vulnerabilities
// and displays the results. It's a simplified version that accepts just the image name and tag.
func Trivy(dockerImage string) error {
	ctx := context.Background()
	args := []string{"image", dockerImage, "--format", "table", "--no-progress"}

	cmd := exec.CommandContext(ctx, "trivy", args...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	fmt.Printf("🔍 Running 'trivy image' scan for %s...\n", dockerImage)

	// Run Trivy scan
	err := cmd.Run()

	outStr := stdoutBuf.String()
	errStr := stderrBuf.String()

	// Handle errors
	if err != nil {
		fmt.Printf("❌ Error running 'trivy image': %v\n", err)
		if errStr != "" {
			fmt.Printf("⚠️ Error details: %s\n", errStr)
		}
		return fmt.Errorf("failed to run 'trivy image': %w", err)
	}

	// Print scan results
	if outStr != "" {
		fmt.Println("📄 Trivy scan results:")
		fmt.Println(outStr)
	}

	fmt.Println("✅ Scan completed successfully.")
	return nil
}

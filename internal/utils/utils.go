package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// CreateYamlFile creates a YAML file with given name and content safely.
func CreateYamlFile(fileName, content string) error {
	filePath := filepath.Join(".", fileName)

	// Prevent overwriting an existing file
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("⚠️  %s already exists. Delete or rename it before creating a new one", fileName)
	}

	// Write YAML content
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("❌ failed to create %s: %v", fileName, err)
	}

	fmt.Printf("✅ %s created successfully at %s\n", fileName, filePath)
	return nil
}

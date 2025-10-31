package configs

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pterm/pterm"
)

// ParseImage splits an image name into its name and tag components.
// If no tag is provided, "latest" is returned.
// used in sdkr package to parse image name and tag
func ParseImage(image string) (string, string, error) {
	parts := strings.SplitN(image, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}
	return image, "latest", nil
}

// ParseEcrImageRef parses an ECR image reference into its account ID, region, repository, and tag components.
// It returns accountID, region, repository, tag, error
func ParseEcrImageRef(imageRef string) (string, string, string, string, error) {
	parts := strings.SplitN(imageRef, ":", 2)
	if len(parts) != 2 {
		pterm.Error.Printfln("Invalid image reference: must be domain/repository:tag")
		return "", "", "", "", errors.New("invalid image reference: must be domain/repository:tag")
	}
	repositoryPart := parts[0]
	tag := parts[1]
	slashParts := strings.SplitN(repositoryPart, "/", 2)
	if len(slashParts) < 2 {
		pterm.Error.Printfln("Invalid image reference: missing repository name (expected domain/repo)")
		return "", "", "", "", fmt.Errorf("invalid image reference: missing repository name (expected domain/repo)")
	}
	domain := slashParts[0]
	repoPath := slashParts[1]

	domainParts := strings.Split(domain, ".")
	if len(domainParts) < 6 {
		pterm.Error.Printfln("Invalid ECR domain format (too few parts): %s", domain)
		return "", "", "", "", fmt.Errorf("invalid ECR domain format (too few parts): %s", domain)
	}

	accountID := domainParts[0]
	region := domainParts[3]
	repository := repoPath

	if accountID == "" || region == "" || repository == "" || tag == "" {
		pterm.Error.Printfln("Missing required ECR parameters in image reference")
		return "", "", "", "", fmt.Errorf("missing required ECR parameters in image reference")
	}

	pterm.Info.Printfln("Successfuly parse ECR image referance...")
	return accountID, region, repository, tag, nil
}

// SplitKeyValue splits a string into two parts at the first occurrence of the "=" character.
// used in selm package to split key value pairs
func SplitKeyValue(arg string) []string {
	parts := make([]string, 2)
	for i, part := range []rune(arg) {
		if part == '=' {
			parts[0] = string([]rune(arg)[:i])
			parts[1] = string([]rune(arg)[i+1:])
			break
		}
	}
	return parts
}

// ParseGhcrImageRef parses a GHCR image reference into its components
func ParseGhcrImageRef(imageRef string) (namespace, repository, tag string, err error) {
	// If it's already a full GHCR reference
	if strings.HasPrefix(imageRef, "ghcr.io/") {
		// Remove the ghcr.io/ prefix
		path := strings.TrimPrefix(imageRef, "ghcr.io/")

		// Split into parts
		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			return "", "", "", fmt.Errorf("invalid GHCR image reference: expected format 'ghcr.io/namespace/repository[:tag]'")
		}

		namespace = parts[0]
		repoWithTag := strings.Join(parts[1:], "/") // Handle nested repositories

		// Split repository and tag
		repoParts := strings.Split(repoWithTag, ":")
		repository = repoParts[0]
		if len(repoParts) > 1 {
			tag = repoParts[1]
		} else {
			tag = "latest"
		}

		return namespace, repository, tag, nil
	}

	// If it's just a local image name, return empty to use flags/config
	return "", "", "", fmt.Errorf("not a GHCR image reference")
}

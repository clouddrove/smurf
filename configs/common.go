package configs

import (
	"fmt"
	"strings"
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
        return "", "", "", "", fmt.Errorf("invalid image reference: must be domain/repository:tag")
    }
    repositoryPart := parts[0]
    tag := parts[1]
    slashParts := strings.SplitN(repositoryPart, "/", 2)
    if len(slashParts) < 2 {
        return "", "", "", "", fmt.Errorf("invalid image reference: missing repository name (expected domain/repo)")
    }
    domain := slashParts[0]
    repoPath := slashParts[1]

    domainParts := strings.Split(domain, ".")
    if len(domainParts) < 6 {
        return "", "", "", "", fmt.Errorf("invalid ECR domain format (too few parts): %s", domain)
    }

    accountID := domainParts[0]
    region := domainParts[3] 
    repository := repoPath

    if accountID == "" || region == "" || repository == "" || tag == "" {
        return "", "", "", "", fmt.Errorf("missing required ECR parameters in image reference")
    }

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

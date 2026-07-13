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

// StripAcrRegistryHost removes an optional ACR hostname from a repository path.
// For example, "myregistry.azurecr.io/my-app" becomes "my-app".
func StripAcrRegistryHost(repository string) string {
	if !strings.Contains(repository, "/") {
		return repository
	}
	parts := strings.SplitN(repository, "/", 2)
	if strings.HasSuffix(parts[0], ".azurecr.io") {
		return parts[1]
	}
	return repository
}

// NormalizeAcrLocalImage converts an image reference to the local Docker image name.
// Full ACR references such as "registry.azurecr.io/my-app:v1" are normalized to "my-app:v1".
func NormalizeAcrLocalImage(imageRef string) (localImage, repository, tag string, err error) {
	repository, tag, err = ParseImage(imageRef)
	if err != nil {
		return "", "", "", err
	}
	if tag == "" {
		tag = "latest"
	}
	repository = StripAcrRegistryHost(repository)
	localImage = fmt.Sprintf("%s:%s", repository, tag)
	return localImage, repository, tag, nil
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

// ParseBuildArgs converts CLI --build-arg values into a key/value map.
// Each flag value can be a single key=value pair or comma-separated pairs:
// --build-arg NODE_ENV=production,API_URL=https://example.com
// Repeated flags are also supported and later values override earlier ones.
func ParseBuildArgs(args []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, raw := range args {
		for _, entry := range splitBuildArgEntries(raw) {
			parts := strings.SplitN(entry, "=", 2)
			if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
				return nil, fmt.Errorf("invalid build-arg %q: expected key=value", entry)
			}
			result[strings.TrimSpace(parts[0])] = parts[1]
		}
	}
	return result, nil
}

func splitBuildArgEntries(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if !strings.Contains(raw, ",") {
		return []string{raw}
	}

	var entries []string
	start := 0
	for i := 0; i < len(raw); i++ {
		if raw[i] != ',' {
			continue
		}

		next := strings.TrimSpace(raw[i+1:])
		eq := strings.Index(next, "=")
		if eq <= 0 {
			continue
		}
		if !isBuildArgKey(strings.TrimSpace(next[:eq])) {
			continue
		}

		entries = append(entries, strings.TrimSpace(raw[start:i]))
		start = i + 1
	}

	entries = append(entries, strings.TrimSpace(raw[start:]))
	return entries
}

func isBuildArgKey(key string) bool {
	if key == "" {
		return false
	}
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_', r == '-':
		default:
			return false
		}
	}
	return true
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

package configs

import "strings"


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

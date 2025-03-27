package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

var (
	version = "v1.0.0"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func getLatestVersion() string {
	url := "https://api.github.com/repos/clouddrove/smurf/releases/latest"
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return version
	}
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return version
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: Unable to fetch latest version. Status code:", resp.StatusCode)
		return version
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return version
	}

	var release githubRelease
	err = json.Unmarshal(body, &release)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return version
	}

	if release.TagName != "" {
		return release.TagName
	}

	return version
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version of your CLI tool",
	Run: func(cmd *cobra.Command, args []string) {
		latestVersion := getLatestVersion()
		fmt.Printf("%s\n", latestVersion)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
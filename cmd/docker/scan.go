package docker

import (
    "github.com/clouddrove/smurf/internal/docker"
    "github.com/pterm/pterm"
    "github.com/spf13/cobra"
)

var dockerTag string
var sarifFile string

var scan = &cobra.Command{
    Use:   "scan",
    Short: "Scan Docker images for known vulnerabilities",
    RunE: func(cmd *cobra.Command, args []string) error {
        err := docker.Scout(dockerTag, sarifFile)
        if err != nil {
            pterm.Error.Println(err)
            return err
        }
        return nil
    },
}

func init() {
    scan.Flags().StringVarP(&dockerTag, "tag", "t", "", "Docker image tag to scan")
    scan.Flags().StringVarP(&sarifFile, "output", "o", "", "Output file for SARIF report")
    scan.MarkFlagRequired("tag")

    sdkrCmd.AddCommand(scan)
}

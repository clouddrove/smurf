package sdkr

import (
	"errors"
	"fmt"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// tagCmd allows you to rename (tag) a Docker image from a specified source to a target reference.
// You can provide both references as command-line arguments or rely on values from the config file.
// If either reference is missing, it attempts to read them from the config.
// On successful tagging, a confirmation message is displayed.
var tagCmd = &cobra.Command{
	Use:          "tag [SOURCE_IMAGE[:TAG]] [TARGET_IMAGE[:TAG]]",
	Short:        "Tag a Docker image for a remote repository",
	Args:         cobra.MaximumNArgs(2),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var source, target string

		if len(args) >= 1 {
			source = args[0]
		}
		if len(args) >= 2 {
			target = args[1]
		}

		if source == "" || target == "" {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			if source == "" {
				source = data.Sdkr.ImageName
			}
			if target == "" {
				target = data.Sdkr.TargetImageTag
			}

			if source == "" || target == "" {
				pterm.Error.Println("both SOURCE and TARGET must be provided either as arguments or in the config")
				return errors.New("both SOURCE and TARGET must be provided either as arguments or in the config")
			}
		}

		pterm.Info.Printf("Tagging image from %q to %q...\n", source, target)
		opts := docker.TagOptions{
			Source: source,
			Target: target,
		}
		if err := docker.TagImage(opts, useAI); err != nil {
			pterm.Error.Printfln("failed to tag image: %v", err)
			return fmt.Errorf("failed to tag image: %v", err)
		}
		pterm.Success.Printf("Successfully tagged image from %q to %q.\n", source, target)
		return nil
	},
	Example: `
  smurf sdkr tag my-app:latest my-org/my-app:prod
  smurf sdkr tag
  # In the second example, it reads SOURCE and TARGET from the config file
`,
}

func init() {
	tagCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")
	sdkrCmd.AddCommand(tagCmd)
}

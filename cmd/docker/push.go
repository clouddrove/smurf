package docker

import (
	"fmt"

	"github.com/spf13/cobra"
)


var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push cmd helps to push images to Docker Hub, ACR, GCR, ECR",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Use 'smurf sdkr push [command]' to push images to Docker Hub, ACR, GCR, ECR ")
		return nil
	},
}

func init() {
	
	sdkrCmd.AddCommand(pushCmd)
}

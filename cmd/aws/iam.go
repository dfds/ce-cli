package aws

import (
	awsCore "github.com/dfds/ce-cli/aws"
	"github.com/spf13/cobra"
)

var IamRoleCreateCmd = &cobra.Command{
	Use:   "create-iam-role",
	Short: "Create IAM role",
	// Long:  `All software has versions. This is Hugo's`,
	Run: func(cmd *cobra.Command, args []string) {
		awsCore.CreateIAMRoleCmd(cmd, args)
	},
}

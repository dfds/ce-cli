package cmd

import (
	"fmt"

	"github.com/dfds/ce-cli/cmd/aws"
	"github.com/spf13/cobra"
)

var awsCmd = &cobra.Command{
	Use:   "aws",
	Short: "Manage resources in AWS accounts",
	// Long:  `All software has versions. This is Hugo's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("aws")
	},
}

func awsInit() {
	// Organizations
	awsCmd.AddCommand(aws.OrgAccountListCmd)

	// IAM
	awsCmd.AddCommand(aws.IamRoleCreateCmd)
}

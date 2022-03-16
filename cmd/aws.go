package cmd

import (
	"github.com/dfds/ce-cli/cmd/aws"
	"github.com/spf13/cobra"
)

var awsCmd = &cobra.Command{
	Use:   "aws",
	Short: "Manage resources in AWS accounts",
	// Long:  `All software has versions. This is Hugo's`,
	Run: func(cmd *cobra.Command, args []string) {},
}

func awsInit() {
	// Emilcat
	awsCmd.PersistentFlags().StringSliceP("include-account-ids", "i", []string{}, "Help text here?")

	// Organizations
	awsCmd.AddCommand(aws.OrgAccountListCmd)

	// IAM
	awsCmd.AddCommand(aws.IamRoleCreateCmd)
	awsCmd.AddCommand(aws.IamRoleDeleteCmd)
}

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
	awsCmd.PersistentFlags().StringSliceP("include-account-ids", "i", []string{}, "Account IDs to target.")
	awsCmd.PersistentFlags().StringSliceP("exclude-account-ids", "e", []string{}, "Account IDs to exclude.")
	awsCmd.PersistentFlags().StringP("path", "t", "/managed/", "The path for the resource.")
	awsCmd.PersistentFlags().Int64P("concurrent-operations", "c", 5, "Maximum number of concurrent operations.")

	// Organizations
	aws.OrgInit()
	awsCmd.AddCommand(aws.OrgAccountListCmd)

	// IAM
	aws.IamInit()
	awsCmd.AddCommand(aws.PredefinedIamRoleCreateCmd)
	awsCmd.AddCommand(aws.IamRoleDeleteCmd)
	awsCmd.AddCommand(aws.IamOIDCProviderCreateCmd)
	awsCmd.AddCommand(aws.IamOIDCProviderDeleteCmd)
	awsCmd.AddCommand(aws.IamOIDCProviderUpdateThumbprintCmd)
}

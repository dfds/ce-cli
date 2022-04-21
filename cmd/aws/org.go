package aws

import (
	awsCore "github.com/dfds/ce-cli/aws"
	"github.com/spf13/cobra"
)

var OrgAccountListCmd = &cobra.Command{
	Use:   "list-org-accounts",
	Short: "List Organization accounts",
	// Long:  `All software has versions. This is Hugo's`,
	Run: func(cmd *cobra.Command, args []string) {
		awsCore.OrgAccountListCmd(cmd, args)
	},
}

func OrgInit() {

	OrgAccountListCmd.PersistentFlags().StringP("bucket-name", "b", "", "The name of the backend S3 bucket for the CLI.")
	OrgAccountListCmd.PersistentFlags().StringP("bucket-role-arn", "", "", "The ARN of the role that will be used to access bucket contents.")
	cobra.MarkFlagRequired(OrgAccountListCmd.PersistentFlags(), "bucket-name")
	cobra.MarkFlagRequired(OrgAccountListCmd.PersistentFlags(), "bucket-role-arn")

}

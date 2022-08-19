package aws

import (
	awsCore "github.com/dfds/ce-cli/aws"
	"github.com/spf13/cobra"
)

var SteamPipeConfigGenerateCmd = &cobra.Command{
	Use:   "steampipe-config",
	Short: "Generate Steampipe config for all available accounts in an AWS organisation",
	Run: func(cmd *cobra.Command, args []string) {
		awsCore.SteamPipeConfigGenerateCmd(cmd, args)
	},
}

var AwsConfigGenerateCmd = &cobra.Command{
	Use:   "aws-config",
	Short: "Generate aws config for all available accounts in an AWS organisation",
	Run: func(cmd *cobra.Command, args []string) {
		awsCore.AwsConfigGenerateCmd(cmd, args)
	},
}

func SteampipeInit() {
	//OrgAccountListCmd.PersistentFlags().StringP("bucket-name", "b", "", "The name of the backend S3 bucket for the CLI.")
	//OrgAccountListCmd.PersistentFlags().StringP("bucket-role-arn", "", "", "The ARN of the role that will be used to access bucket contents.")
	//cobra.MarkFlagRequired(OrgAccountListCmd.PersistentFlags(), "bucket-name")
	//cobra.MarkFlagRequired(OrgAccountListCmd.PersistentFlags(), "bucket-role-arn")

}

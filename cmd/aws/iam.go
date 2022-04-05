package aws

import (
	awsCore "github.com/dfds/ce-cli/aws"
	"github.com/spf13/cobra"
)

var PredefinedIamRoleCreateCmd = &cobra.Command{
	Use:   "create-predefined-iam-role",
	Short: "Create predefined IAM role",
	Run: func(cmd *cobra.Command, args []string) {
		awsCore.CreatePredefinedIAMRoleCmd(cmd, args)
	},
}

var PredefinedIamRoleDeleteCmd = &cobra.Command{
	Use:   "delete-predefined-iam-role",
	Short: "Delete predefined IAM role",
	Run: func(cmd *cobra.Command, args []string) {
		awsCore.DeletePredefinedIAMRoleCmd(cmd, args)
	},
}

func IamInit() {

	PredefinedIamRoleCreateCmd.PersistentFlags().StringP("role-name", "r", "", "The name of a unique predefined role that will be deployed into the accounts specified.")
	PredefinedIamRoleCreateCmd.PersistentFlags().StringP("bucket-name", "b", "", "The name of an S3 Bucket where the Policy and Trust documents are held.")
	PredefinedIamRoleCreateCmd.PersistentFlags().StringP("bucket-role-arn", "", "", "The ARN of the role that will be used to access bucket contents.")

	// set mandatory parameter requirements for the command
	cobra.MarkFlagRequired(PredefinedIamRoleCreateCmd.PersistentFlags(), "role-name")
	cobra.MarkFlagRequired(PredefinedIamRoleCreateCmd.PersistentFlags(), "bucket-name")
	cobra.MarkFlagRequired(PredefinedIamRoleCreateCmd.PersistentFlags(), "bucket-role-arn")

	PredefinedIamRoleDeleteCmd.PersistentFlags().StringP("role-name", "r", "", "The name of the role to be deleted.")

}

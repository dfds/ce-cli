package aws

import (
	awsCore "github.com/dfds/ce-cli/aws"
	"github.com/spf13/cobra"
)

var IamOIDCProviderUpdateThumbprintCmd = &cobra.Command{
	Use:   "update-oidc-provider-thumbprint",
	Short: "Updates the thumbprint associated with an IAM Open ID Connect Provider",
	Run: func(cmd *cobra.Command, args []string) {
		awsCore.UpdateIAMOIDCProviderThumbprintCmd(cmd, args)
	},
}

var IamOIDCProviderCreateCmd = &cobra.Command{
	Use:   "create-oidc-provider",
	Short: "Create an IAM Open ID Connect Provider",
	Run: func(cmd *cobra.Command, args []string) {
		awsCore.CreateIAMOIDCProviderCmd(cmd, args)
	},
}

var IamOIDCProviderDeleteCmd = &cobra.Command{
	Use:   "delete-oidc-provider",
	Short: "Delete an IAM Open ID Connect Provider",
	Run: func(cmd *cobra.Command, args []string) {
		awsCore.DeleteIAMOIDCProviderCmd(cmd, args)
	},
}

var PredefinedIamRoleCreateCmd = &cobra.Command{
	Use:   "create-predefined-iam-role",
	Short: "Create predefined IAM role",
	Run: func(cmd *cobra.Command, args []string) {
		awsCore.CreatePredefinedIAMRoleCmd(cmd, args)
	},
}

var IamRoleDeleteCmd = &cobra.Command{
	Use:   "delete-iam-role",
	Short: "Delete an IAM role",
	Run: func(cmd *cobra.Command, args []string) {
		awsCore.DeleteIAMRoleCmd(cmd, args)
	},
}

func IamInit() {

	PredefinedIamRoleCreateCmd.PersistentFlags().StringP("role-name", "r", "", "The name of a unique predefined role that will be deployed into the accounts specified.")
	PredefinedIamRoleCreateCmd.PersistentFlags().StringP("bucket-name", "b", "", "The name of an S3 Bucket where the Policy and Trust documents are held.")
	PredefinedIamRoleCreateCmd.PersistentFlags().StringP("bucket-role-arn", "", "", "The ARN of the role that will be used to access bucket contents.")
	cobra.MarkFlagRequired(PredefinedIamRoleCreateCmd.PersistentFlags(), "role-name")
	cobra.MarkFlagRequired(PredefinedIamRoleCreateCmd.PersistentFlags(), "bucket-name")
	cobra.MarkFlagRequired(PredefinedIamRoleCreateCmd.PersistentFlags(), "bucket-role-arn")

	IamRoleDeleteCmd.PersistentFlags().StringP("role-name", "r", "", "The name of the role to be deleted.")
	cobra.MarkFlagRequired(IamRoleDeleteCmd.PersistentFlags(), "role-name")

	IamOIDCProviderCreateCmd.PersistentFlags().StringP("url", "u", "", "The URL for the OpenID Connect provider.")
	IamOIDCProviderCreateCmd.PersistentFlags().StringP("cluster-name", "", "", "The name of the Cluster targetted by the OpenID Connect provider.")
	cobra.MarkFlagRequired(IamOIDCProviderCreateCmd.PersistentFlags(), "url")
	cobra.MarkFlagRequired(IamOIDCProviderCreateCmd.PersistentFlags(), "cluster-name")

	IamOIDCProviderDeleteCmd.PersistentFlags().StringP("url", "u", "", "The URL for the OpenID Connect provider.")
	cobra.MarkFlagRequired(IamOIDCProviderDeleteCmd.PersistentFlags(), "url")

	IamOIDCProviderUpdateThumbprintCmd.PersistentFlags().StringP("url", "u", "", "The URL for the OpenID Connect provider.")
	cobra.MarkFlagRequired(IamOIDCProviderUpdateThumbprintCmd.PersistentFlags(), "url")

}

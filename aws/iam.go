package aws

import (
	"context"
	"fmt"
	"log"

	// "github.com/dfds/ce-cli/aws"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"

	"github.com/spf13/cobra"
)

func CreateIAMRoleCmd(cmd *cobra.Command, args []string) {

	includeAccountIds, _ := cmd.Flags().GetStringSlice("include-account-ids")

	accounts := OrgAccountList(includeAccountIds)
	var ids []string
	for _, v := range accounts {
		ids = append(ids, *v.Id)
	}
	assumedRoles := AssumeRoleMultipleAccounts(ids)

	for _, creds := range assumedRoles {

		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(*creds.AccessKeyId, *creds.SecretAccessKey, *creds.SessionToken)), config.WithRegion("eu-west-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}

		assumedClient := iam.NewFromConfig(cfg)

		CreateIAMRole(assumedClient)
		// fmt.Println(accountId, creds)
	}

}

func CreateIAMRole(client *iam.Client) {

	resp, err := client.GetAccountSummary(context.TODO(), nil)
	if err == nil {
		fmt.Printf("resp: %v\n", resp)
	}

}

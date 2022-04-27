package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"

	"github.com/spf13/cobra"
)

func OrgAccountListCmd(cmd *cobra.Command, args []string) {

	includeAccountIds, _ := cmd.Flags().GetStringSlice("include-account-ids")
	excludeAccountIds, _ := cmd.Flags().GetStringSlice("exclude-account-ids")

	bucketName, _ := cmd.Flags().GetString("bucket-name")
	bucketRoleArn, _ := cmd.Flags().GetString("bucket-role-arn")

	// Merge always excluded account IDs from backend bucket, with those supplied as args
	excludeAccountIdsS3 := GetExcludeAccountIdsFromS3(bucketName, bucketRoleArn, "aws/org/excludeAccountIds.json", "ListAccounts")
	excludeAccountIds = append(excludeAccountIds, excludeAccountIdsS3...)

	accountList, err := OrgAccountList(includeAccountIds, excludeAccountIds)
	if err != nil {
		fmt.Printf("Errr: %s\n", err)
	} else {
		for _, v := range accountList {
			fmt.Println(*v.Id)
		}
	}
}

func OrgAccountList(includeAccountIds []string, excludeAccountIds []string) ([]types.Account, error) {

	// try to create a default config instance
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-west-1"))

	// in the event of an issue return nil output and the error; it's expected that the caller will handle
	// the error
	if err != nil {
		return nil, err
	}

	svc := organizations.NewFromConfig(cfg)

	var accountList []types.Account

	accountPage, err := svc.ListAccounts(context.TODO(), &organizations.ListAccountsInput{NextToken: nil})
	if err != nil {
		return nil, err
	}
	accountList = append(accountList, accountPage.Accounts...)

	for accountPage.NextToken != nil {
		accountPage, err = svc.ListAccounts(context.TODO(), &organizations.ListAccountsInput{NextToken: accountPage.NextToken})
		if err != nil {
			return nil, err
		}
		accountList = append(accountList, accountPage.Accounts...)
	}

	// Filter account list to "included account ids""
	if len(includeAccountIds) > 0 {
		var includedAccountList []types.Account
		for _, v := range accountList {
			for _, incId := range includeAccountIds {
				if *v.Id == incId {
					includedAccountList = append(includedAccountList, v)
				}
			}
		}
		accountList = includedAccountList
	}

	// Remove any excluded account ids
	if len(excludeAccountIds) > 0 {
		var filteredAccountList []types.Account
		var excluded bool
		for _, v := range accountList {
			excluded = false
			for _, exclId := range excludeAccountIds {
				if *v.Id == exclId {
					excluded = true
				}
			}
			if excluded == false {
				filteredAccountList = append(filteredAccountList, v)
			}
		}
		accountList = filteredAccountList
	}

	// return the accounts list and no error
	return accountList, nil
}

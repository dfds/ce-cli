package aws

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"

	"github.com/spf13/cobra"
)

func OrgAccountListCmd(cmd *cobra.Command, args []string) {
	for _, v := range OrgAccountList() {
		fmt.Println(*v.Id)
	}
}

func OrgAccountList() []types.Account {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-west-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	svc := organizations.NewFromConfig(cfg)

	var accountList []types.Account

	accountPage, err := svc.ListAccounts(context.TODO(), &organizations.ListAccountsInput{NextToken: nil})
	if err != nil {
		log.Fatalf("unable to list accounts, %v", err)
	}
	accountList = append(accountList, accountPage.Accounts...)

	for accountPage.NextToken != nil {
		accountPage, err = svc.ListAccounts(context.TODO(), &organizations.ListAccountsInput{NextToken: accountPage.NextToken})
		if err != nil {
			log.Fatalf("unable to list accounts, %v", err)
		}
		accountList = append(accountList, accountPage.Accounts...)
	}

	return accountList

}

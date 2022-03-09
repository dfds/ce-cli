package aws

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/organizations"

	"github.com/spf13/cobra"
)

func OrgAccountListCmd(cmd *cobra.Command, args []string) {
	OrgAccountList()
}

func OrgAccountList() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-west-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	svc := organizations.NewFromConfig(cfg)

	accounts, err := svc.ListAccounts(context.TODO(), &organizations.ListAccountsInput{})
	if err != nil {
		log.Fatalf("unable to list accounts, %v", err)
	}

	fmt.Println(accounts.Accounts)

}

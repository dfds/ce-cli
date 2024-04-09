package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	types2 "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/spf13/cobra"
	"log"
	"os"
	"reflect"
)

var UpdateEcrTrustedAccountsCmd = &cobra.Command{
	Use:   "ecr",
	Short: "Update trust policy in all ECR repos",
	Run: func(cmd *cobra.Command, args []string) {
		UpdateEcrTrustedAccounts(cmd, args)
	},
}

func UpdateEcrTrustedAccounts(cmd *cobra.Command, args []string) {
	//url, _ := cmd.Flags().GetString("url")
	//includeAccountIds, _ := cmd.Flags().GetStringSlice("include-account-ids")
	//excludeAccountIds, _ := cmd.Flags().GetStringSlice("exclude-account-ids")
	//concurrentOps, _ := cmd.Flags().GetInt64("concurrent-operations")
	//
	//var waitGroup sync.WaitGroup
	//sem := semaphore.NewWeighted(concurrentOps)
	ctx := context.Background()
	//startTime := time.Now()

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-central-1"))
	if err != nil {
		log.Fatal(err)
	}
	client := ecr.NewFromConfig(cfg)

	pag := ecr.NewDescribeRepositoriesPaginator(client, &ecr.DescribeRepositoriesInput{MaxResults: aws.Int32(700)})
	repos := []types2.Repository{}
	for pag.HasMorePages() {
		resp, err := pag.NextPage(ctx)
		if err != nil {
			log.Fatal(err)
		}
		repos = append(repos, resp.Repositories...)
	}
	counter := 0

	for _, repo := range repos {
		policyResp, err := client.GetRepositoryPolicy(ctx, &ecr.GetRepositoryPolicyInput{
			RepositoryName: repo.RepositoryName,
		})
		if err != nil {
			log.Println(err)
			continue
		}

		if policyResp.PolicyText == nil {
			fmt.Printf("Policy for repo '%s' doesn't exist, skipping\n", *repo.RepositoryName)
			continue
		}

		var parsedPolicy ecrPolicyJson
		err = json.Unmarshal([]byte(*policyResp.PolicyText), &parsedPolicy)
		if err != nil {
			fmt.Println(*repo.RepositoryName)
			log.Fatal(err)
		}

		if len(parsedPolicy.Statement) != 1 {
			fmt.Printf("Policy for repo '%s' looks different, skipping\n", *repo.RepositoryName)
			continue
		}

		principalType := reflect.TypeOf(parsedPolicy.Statement[0].Principal)

		//fmt.Printf("type: '%s'\n", principalType.String())
		switch principalType.String() {
		case "string":
			fmt.Printf("principal: %s\n", parsedPolicy.Statement[0].Principal.(string))
		case "map[string]interface {}":
			principalMap := parsedPolicy.Statement[0].Principal.(map[string]interface{})
			if val, ok := principalMap["AWS"]; ok {
				innerType := reflect.TypeOf(val)
				if innerType.String() == "[]interface {}" {
					fmt.Printf("inner type: '%s'\n", innerType.String())
					accountIdsCasted := []string{}
					for _, id := range val.([]interface{}) {
						accountIdsCasted = append(accountIdsCasted, id.(string))
					}

					fmt.Println(accountIdsCasted)
					counter = counter + 1
				}
			}
		}

		//accountIds := parsedPolicy.Statement[0].Principal.AWS
		//fmt.Println(accountIds)
	}

	fmt.Println(counter)
	os.Exit(1)
}

type ecrPolicyJson struct {
	Version   string `json:"Version"`
	Statement []struct {
		Sid       string      `json:"Sid"`
		Effect    string      `json:"Effect"`
		Principal interface{} `json:"Principal"`
		Action    []string    `json:"Action"`
	} `json:"Statement"`
}

type PrincipalString struct {
	AWS string `json:"AWS"`
}

type PrincipalArray struct {
	AWS string `json:"AWS"`
}

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

					if isStringMissingInArray(accountIdsCasted, "arn:aws:iam::ID:root") {
						fmt.Println("'ID' is missing in trust, adding")
						newAccountIdsArray := []string{}
						newAccountIdsArray = append(newAccountIdsArray, accountIdsCasted...)
						newAccountIdsArray = append(newAccountIdsArray, "arn:aws:iam::ID:root")

						updatedPolicy := cloneEcrPolicy(parsedPolicy)
						upPrincipal := make(map[string]interface{})
						upPrincipal["AWS"] = newAccountIdsArray
						updatedPolicy.Statement[0].Principal = upPrincipal

						//preUpdate, err := json.MarshalIndent(parsedPolicy, "", "  ")
						//if err != nil {
						//	log.Fatal(err)
						//}
						//
						postUpdate, err := json.MarshalIndent(updatedPolicy, "", "  ")
						if err != nil {
							log.Fatal(err)
						}
						postUpdateString := string(postUpdate)

						fmt.Printf("repo name: %s\n", *policyResp.RepositoryName)

						_, err = client.SetRepositoryPolicy(ctx, &ecr.SetRepositoryPolicyInput{
							RepositoryName: policyResp.RepositoryName,
							PolicyText:     &postUpdateString,
						})
						if err != nil {
							log.Fatal(err)
						}

						//fmt.Printf("ogJson:\n%s\n", *policyResp.PolicyText)
						//fmt.Printf("pre:\n%s\n", preUpdate)
						//fmt.Printf("post:\n%s\n", postUpdate)
						//fmt.Printf("----------\n\n")
					} else {
						continue
					}

					counter = counter + 1

					//if counter == 3 {
					//	os.Exit(1)
					//}
				}
			}
		}

	}

	fmt.Println(counter)
	os.Exit(1)
}

func isStringMissingInArray(data []string, compr string) bool {
	for _, val := range data {
		if val == compr {
			return false
		}
	}
	return true
}

func cloneEcrPolicy(og ecrPolicyJson) ecrPolicyJson {
	newObj := ecrPolicyJson{
		Version:   og.Version,
		Statement: make([]ecrPolicyJsonStatement, len(og.Statement)),
	}

	_ = copy(newObj.Statement, og.Statement)
	return newObj
}

type ecrPolicyJson struct {
	Version   string                   `json:"Version"`
	Statement []ecrPolicyJsonStatement `json:"Statement"`
}

type ecrPolicyJsonStatement struct {
	Sid       string      `json:"Sid"`
	Effect    string      `json:"Effect"`
	Principal interface{} `json:"Principal"`
	Action    []string    `json:"Action"`
}

type PrincipalString struct {
	AWS string `json:"AWS"`
}

type PrincipalArray struct {
	AWS string `json:"AWS"`
}

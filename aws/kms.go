package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/sync/semaphore"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

var GetKmsKeysCmd = &cobra.Command{
	Use:   "kms",
	Short: "Get KMS keys across all org accounts",
	Run: func(cmd *cobra.Command, args []string) {
		GetKmsKeys(cmd, args)
	},
}

func GetKmsKeys(cmd *cobra.Command, args []string) {
	url, _ := cmd.Flags().GetString("url")
	includeAccountIds, _ := cmd.Flags().GetStringSlice("include-account-ids")
	excludeAccountIds, _ := cmd.Flags().GetStringSlice("exclude-account-ids")
	concurrentOps, _ := cmd.Flags().GetInt64("concurrent-operations")

	var waitGroup sync.WaitGroup
	sem := semaphore.NewWeighted(concurrentOps)
	ctx := context.TODO()
	startTime := time.Now()

	targetAccounts := make(map[string]string)

	// get list of org accounts
	color.Set(color.FgWhite)
	fmt.Printf("Obtaining a list of Organizational Accounts: ")
	accounts, err := OrgAccountList(includeAccountIds, excludeAccountIds)
	if err != nil {
		color.Red("Failed")
		color.Yellow("  Error: %v", err)
		os.Exit(1)
	} else {
		for _, v := range accounts {
			targetAccounts[*v.Id] = *v.Name
		}
		color.Green("Done")
	}

	// assume roles in org accounts
	assumedRoles := AssumeRoleMultipleAccounts(targetAccounts)
	var keysList []keyData
	//var regions []string = []string{"eu-west-1", "eu-central-1", "eu-west-2", "us-east-1", "us-east-2"}
	regions, err := GetRegions(ctx)
	if err != nil {
		log.Fatal(err)
	}
	keysLock := &sync.Mutex{}

	for id, creds := range assumedRoles {

		waitGroup.Add(1)

		go func(id string, creds *ststypes.Credentials, url string) {

			color.Set(color.FgWhite)
			fmt.Printf(" Account %s (%s): Finding KMS crypto keys.\n", targetAccounts[id], id)
			sem.Acquire(ctx, 1)
			defer sem.Release(1)
			defer waitGroup.Done()

			for _, region := range regions {
				cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(*creds.AccessKeyId, *creds.SecretAccessKey, *creds.SessionToken)), config.WithRegion(region))
				if err != nil {
					log.Fatalf("unable to load SDK config, %v", err)
				}

				// get a new client used the config we just generated
				var limit int32 = 999
				assumedClient := kms.NewFromConfig(cfg)

				resp, err := assumedClient.ListKeys(ctx, &kms.ListKeysInput{
					Limit: &limit,
				})
				if err == nil {
					if len(resp.Keys) > 0 {
						for _, key := range resp.Keys {
							keyResp, err := assumedClient.DescribeKey(ctx, &kms.DescribeKeyInput{KeyId: key.KeyId})
							if err != nil {
								if strings.Contains(err.Error(), "is not authorized to perform") {
									continue
								}
								log.Fatal(err)
							}

							keyAliases, err := assumedClient.ListAliases(ctx, &kms.ListAliasesInput{KeyId: key.KeyId})
							if err != nil {
								log.Fatal(err)
							}
							keyName := ""

							if len(keyAliases.Aliases) > 0 {
								keyName = *keyAliases.Aliases[0].AliasName
							}

							{
								keysLock.Lock()
								keysList = append(keysList, keyData{
									AwsAccountId: id,
									KeyId:        *key.KeyId,
									KeyArn:       *key.KeyArn,
									Enabled:      keyResp.KeyMetadata.Enabled,
									Name:         keyName,
									KeyManager:   string(keyResp.KeyMetadata.KeyManager),
								})
								keysLock.Unlock()
							}
						}

					}
				} else {
					log.Fatal(err)
				}
			}

		}(id, creds, url)
	}

	waitGroup.Wait()

	fmt.Println("accountId,keyId,keyArn,enabled,name,type,keyManager")
	for _, kmsKey := range keysList {
		fmt.Printf("%s,%s,%s,%t,%s,%s,%s\n", kmsKey.AwsAccountId, kmsKey.KeyId, kmsKey.KeyArn, kmsKey.Enabled, kmsKey.Name, kmsKey.Type, kmsKey.KeyManager)
	}

	color.Set(color.FgCyan)
	fmt.Printf("\nTook %f seconds to gather all KMS keys.\n", time.Since(startTime).Seconds())
	color.Unset()

}

type keyData struct {
	AwsAccountId string
	KeyId        string
	KeyArn       string
	Enabled      bool
	Name         string
	Type         string
	KeyManager   string
}

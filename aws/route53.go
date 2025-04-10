package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/sync/semaphore"
	"log"
	"os"
	"sync"
	"time"
)

var GetRoute53Cmd = &cobra.Command{
	Use:   "route53",
	Short: "Get route53 zones across org accounts",
	Run: func(cmd *cobra.Command, args []string) {
		GetRoute53Zones(cmd, args)
	},
}

func GetRoute53Zones(cmd *cobra.Command, args []string) {
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
	//var instanceList []ec2InstanceData
	//var regions []string = []string{"eu-west-1", "eu-central-1", "eu-west-2", "us-east-1", "us-east-2"}
	//keysLock := &sync.Mutex{}

	for id, creds := range assumedRoles {

		waitGroup.Add(1)

		go func(id string, creds *ststypes.Credentials, url string) {

			color.Set(color.FgWhite)
			fmt.Printf(" Account %s (%s): Finding route53 hosted zones.\n", targetAccounts[id], id)
			sem.Acquire(ctx, 1)
			defer sem.Release(1)
			defer waitGroup.Done()

			cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(*creds.AccessKeyId, *creds.SecretAccessKey, *creds.SessionToken)), config.WithRegion("us-east-1"))
			if err != nil {
				log.Fatalf("unable to load SDK config, %v", err)
			}
			route53Client := route53.NewFromConfig(cfg)

			resp, err := route53Client.ListHostedZones(ctx, &route53.ListHostedZonesInput{})
			if err != nil {
				log.Println(fmt.Sprintf("Unable to list hosted zones for account %s", id))
			} else {
				for _, zone := range resp.HostedZones {
					fmt.Printf("account id: %s, zone name: %s\n", id, *zone.Name)
				}
			}

		}(id, creds, url)
	}

	waitGroup.Wait()

	//fmt.Println("accountId,accountAlias,region,instanceId,launchTime,platformDetails,amiId,amiName,amiOwner,amiCreationTime,AmiDeprecationTime,AmiDescription")
	//for _, instance := range instanceList {
	//	fmt.Printf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n", instance.AwsAccountId, instance.AwsAccountAlias, instance.Region, instance.InstanceId, instance.LaunchTime, instance.PlatformDetails, instance.AmiId, instance.AmiName, instance.AmiOwner, instance.AmiCreationTime, instance.AmiDeprecationTime, instance.AmiDescription)
	//}

	color.Set(color.FgCyan)
	fmt.Printf("\nTook %f seconds to gather all EC2 instances.\n", time.Since(startTime).Seconds())
	color.Unset()

}

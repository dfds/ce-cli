package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
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

var GetEc2InstancesCmd = &cobra.Command{
	Use:   "ec2",
	Short: "Get EC2 instances across all org accounts",
	Run: func(cmd *cobra.Command, args []string) {
		GetEC2Instances(cmd, args)
	},
}

func GetEC2Instances(cmd *cobra.Command, args []string) {
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
	var instanceList []ec2InstanceData
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
			fmt.Printf(" Account %s (%s): Finding EC2 instances.\n", targetAccounts[id], id)
			sem.Acquire(ctx, 1)
			defer sem.Release(1)
			defer waitGroup.Done()

			for _, region := range regions {
				cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(*creds.AccessKeyId, *creds.SecretAccessKey, *creds.SessionToken)), config.WithRegion(region))
				if err != nil {
					log.Fatalf("unable to load SDK config, %v", err)
				}

				ec2Client := ec2.NewFromConfig(cfg)
				pag := ec2.NewDescribeInstancesPaginator(ec2Client, &ec2.DescribeInstancesInput{})
				data := []types.Reservation{}
				for pag.HasMorePages() {
					resp, err := pag.NextPage(ctx)
					if err != nil {
						break
					}
					data = append(data, resp.Reservations...)
				}

				for _, reservation := range data {
					for _, instance := range reservation.Instances {
						payload := ec2InstanceData{
							InstanceId:      *instance.InstanceId,
							Region:          region,
							AwsAccountId:    id,
							AwsAccountAlias: targetAccounts[id],
							LaunchTime:      instance.LaunchTime.String(),
							PlatformDetails: *instance.PlatformDetails,
						}

						amiId := instance.ImageId
						imagesResp, err := ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
							ImageIds: []string{*amiId},
						})
						if err != nil {
							log.Fatal(err)
						}

						if len(imagesResp.Images) > 0 {
							image := imagesResp.Images[0]
							payload.AmiId = *image.ImageId
							if image.Description != nil {
								payload.AmiDescription = strings.ReplaceAll(*image.Description, ",", ";")
							}
							if image.ImageOwnerAlias != nil {
								payload.AmiOwner = *image.ImageOwnerAlias
							}
							if image.CreationDate != nil {
								payload.AmiCreationTime = *image.CreationDate
							}
							if image.DeprecationTime != nil {
								payload.AmiDeprecationTime = *image.DeprecationTime
							}
						}

						keysLock.Lock()
						instanceList = append(instanceList, payload)
						keysLock.Unlock()
					}
				}
			}

		}(id, creds, url)
	}

	waitGroup.Wait()

	fmt.Println("accountId,accountAlias,region,instanceId,launchTime,platformDetails,amiId,amiName,amiOwner,amiCreationTime,AmiDeprecationTime,AmiDescription")
	for _, instance := range instanceList {
		fmt.Printf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n", instance.AwsAccountId, instance.AwsAccountAlias, instance.Region, instance.InstanceId, instance.LaunchTime, instance.PlatformDetails, instance.AmiId, instance.AmiName, instance.AmiOwner, instance.AmiCreationTime, instance.AmiDeprecationTime, instance.AmiDescription)
	}

	color.Set(color.FgCyan)
	fmt.Printf("\nTook %f seconds to gather all EC2 instances.\n", time.Since(startTime).Seconds())
	color.Unset()

}

type ec2InstanceData struct {
	// instance
	AwsAccountId    string
	AwsAccountAlias string
	Region          string
	InstanceId      string
	LaunchTime      string
	PlatformDetails string
	// ami
	AmiId              string
	AmiName            string
	AmiOwner           string
	AmiArchitecture    string
	AmiCreationTime    string
	AmiDeprecationTime string
	AmiDescription     string
}

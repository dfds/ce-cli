package aws

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
)

func AssumeRole(roleArn string) (*types.Credentials, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-west-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	stsClient := sts.NewFromConfig(cfg)

	roleSessionName := "TBD"

	assumedRole, err := stsClient.AssumeRole(context.TODO(), &sts.AssumeRoleInput{RoleArn: &roleArn, RoleSessionName: &roleSessionName})
	if err != nil {
		log.Println("unable to assume role %s, %v", roleArn, err)
		return nil, err
	}

	return assumedRole.Credentials, nil

}

func AssumeRoleMultipleAccounts(accountIds []string) {

	for _, accountId := range accountIds {
		roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, "OrgRole")
		role, _ := AssumeRole(roleArn)
		if role != nil {
			fmt.Println(*role.AccessKeyId)
		}
	}

}

package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/dfds/ce-cli/util"
	"log"
)

func AssumeRole(roleArn string) (*types.Credentials, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-west-1"), config.WithHTTPClient(util.CreateHttpClientWithoutKeepAlive()))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	stsClient := sts.NewFromConfig(cfg)

	roleSessionName := "TBD"

	assumedRole, err := stsClient.AssumeRole(context.TODO(), &sts.AssumeRoleInput{RoleArn: &roleArn, RoleSessionName: &roleSessionName})
	if err != nil {
		log.Printf("unable to assume role %s, %v", roleArn, err)
		return nil, err
	}

	return assumedRole.Credentials, nil

}

func AssumeRoleMultipleAccounts(accounts map[string]string) map[string]*types.Credentials {

	assumedRoles := make(map[string]*types.Credentials)

	for id, _ := range accounts {
		roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", id, "OrgRole")
		role, err := AssumeRole(roleArn)
		if err != nil {
			fmt.Printf("Role Assummption Error: %v\n", err)
		}
		if role != nil {
			assumedRoles[id] = role
		}
	}
	return assumedRoles
}

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	// "github.com/dfds/ce-cli/aws"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"golang.org/x/sync/semaphore"

	"github.com/spf13/cobra"
)

func DefaultTags() []types.Tag {

	var tags []types.Tag

	tags = append(tags, types.Tag{
		Key:   createString("managedBy"),
		Value: createString("ce-cli"),
	})

	tags = append(tags, types.Tag{
		Key:   createString("lastUpdated"),
		Value: createString(time.Now().UTC().Format(time.RFC3339)),
	})

	return tags
}

func DetachRolePolicies(client *iam.Client, name string) {

	attachedPolicies, err := client.ListAttachedRolePolicies(context.TODO(), &iam.ListAttachedRolePoliciesInput{RoleName: &name})

	if err != nil {
		fmt.Printf("Error listing attached Role Policies: %v\n", err)
	} else {
		for _, v := range attachedPolicies.AttachedPolicies {
			client.DetachRolePolicy(context.TODO(), &iam.DetachRolePolicyInput{
				PolicyArn: v.PolicyArn,
				RoleName:  &name,
			})
		}
	}
}

func DeleteIAMRoleCmd(cmd *cobra.Command, args []string) {

	// get parameters from Cobra
	includeAccountIds, _ := cmd.Flags().GetStringSlice("include-account-ids")
	concurrentOps, _ := cmd.Flags().GetInt64("concurrent-operations")

	var waitGroup sync.WaitGroup
	sem := semaphore.NewWeighted(concurrentOps)
	ctx := context.TODO()
	startTime := time.Now()

	accounts := OrgAccountList(includeAccountIds)
	var ids []string
	for _, v := range accounts {
		ids = append(ids, *v.Id)
	}
	assumedRoles := AssumeRoleMultipleAccounts(ids)

	for id, creds := range assumedRoles {

		waitGroup.Add(1)

		go func(id string, creds *ststypes.Credentials) {

			fmt.Printf("Deleting Role in Account %s\n", id)
			sem.Acquire(ctx, 1)
			defer sem.Release(1)
			defer waitGroup.Done()

			cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(*creds.AccessKeyId, *creds.SecretAccessKey, *creds.SessionToken)), config.WithRegion("eu-west-1"))
			if err != nil {
				log.Fatalf("unable to load SDK config, %v", err)
			}

			assumedClient := iam.NewFromConfig(cfg)

			DeleteIAMRole(assumedClient)
			DeleteIAMPolicy(assumedClient)
			fmt.Printf("Role deletion complete in Account %s\n", id)
		}(id, creds)
	}

	waitGroup.Wait()

	fmt.Printf("Took %f seconds to complete deletion.", time.Since(startTime).Seconds())
}

func CreateIAMRoleCmd(cmd *cobra.Command, args []string) {

	// get parameters from Cobra
	includeAccountIds, _ := cmd.Flags().GetStringSlice("include-account-ids")
	concurrentOps, _ := cmd.Flags().GetInt64("concurrent-operations")

	var waitGroup sync.WaitGroup
	sem := semaphore.NewWeighted(concurrentOps)
	ctx := context.TODO()
	startTime := time.Now()

	accounts := OrgAccountList(includeAccountIds)
	var ids []string
	for _, v := range accounts {
		ids = append(ids, *v.Id)
	}
	assumedRoles := AssumeRoleMultipleAccounts(ids)

	for id, creds := range assumedRoles {

		waitGroup.Add(1)

		go func(id string, creds *ststypes.Credentials) {

			fmt.Printf("Creating Role in Account %s\n", id)
			sem.Acquire(ctx, 1)
			defer sem.Release(1)
			defer waitGroup.Done()

			cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(*creds.AccessKeyId, *creds.SecretAccessKey, *creds.SessionToken)), config.WithRegion("eu-west-1"))
			if err != nil {
				log.Fatalf("unable to load SDK config, %v", err)
			}

			assumedClient := iam.NewFromConfig(cfg)

			inventoryPolicy := CreateIAMPolicy(assumedClient)
			CreateIAMRole(assumedClient)
			AttachIAMPolicy(assumedClient, "arn:aws:iam::aws:policy/job-function/ViewOnlyAccess", "Inventory")
			AttachIAMPolicy(assumedClient, *inventoryPolicy.Policy.Arn, "Inventory")
			fmt.Printf("Role creation complete in Account %s\n", id)
		}(id, creds)
	}

	waitGroup.Wait()

	fmt.Printf("Took %f seconds to complete creation.", time.Since(startTime).Seconds())
}

func CreateIAMPolicy(client *iam.Client) *iam.CreatePolicyOutput {

	policyDocument := `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "CloudEngineeringCLI",
            "Effect": "Allow",
            "Action": "s3:ListBucket",
            "Resource": "arn:aws:s3:::mynewbucket"
        }
    ]
}`
	policyName := "Inventory"
	path := "/managed/"
	description := "Additional policy for doing inventory"

	var input *iam.CreatePolicyInput = &iam.CreatePolicyInput{
		PolicyName:     &policyName,
		Path:           &path,
		Description:    &description,
		PolicyDocument: &policyDocument,
	}
	resp, err := client.CreatePolicy(context.TODO(), input)

	if err != nil {
		var eae *types.EntityAlreadyExistsException
		if errors.As(err, &eae) {

			log.Printf("Warning: Policy '%s' already exists\n", policyName)

			// Get existing policy ARN
			managedPolicies, err := client.ListPolicies(context.TODO(), &iam.ListPoliciesInput{
				PathPrefix: &path,
			})
			if err != nil {
				fmt.Printf("Could not list policies: %v\n", err)
			}
			var policyArn *string
			for _, v := range managedPolicies.Policies {
				if *v.Path == path && *v.PolicyName == policyName {
					policyArn = v.Arn
				}
			}

			// Compare policy documents
			policy, err := client.GetPolicy(context.TODO(), &iam.GetPolicyInput{PolicyArn: policyArn})
			if err != nil {
				fmt.Printf("Cannot get policy %s: %v\n", *policyArn, err)
			}
			policyVersion := policy.Policy.DefaultVersionId

			policyContent, err := client.GetPolicyVersion(context.TODO(), &iam.GetPolicyVersionInput{
				PolicyArn: policyArn,
				VersionId: policyVersion,
			})
			if err != nil {
				fmt.Printf("Coult not get policy content: %v\n", err)
			}

			currentPolicy, err := url.QueryUnescape(*policyContent.PolicyVersion.Document)
			if err != nil {
				fmt.Printf("Could not decode policy document: %v\n", err)
			}
			if currentPolicy != policyDocument {
				_, err := client.CreatePolicyVersion(context.TODO(), &iam.CreatePolicyVersionInput{
					PolicyArn:      policyArn,
					PolicyDocument: &policyDocument,
					SetAsDefault:   true,
				})
				if err != nil {
					fmt.Printf("Could not create policy version: %v\n", err)
				}

			}

			// Generate policy response
			resp = &iam.CreatePolicyOutput{
				Policy: &types.Policy{
					Arn: policyArn,
				},
			}

		} else {
			fmt.Printf("err: %v\n", err)
		}
	}

	// ensure tags on the policy are replaced with those provided
	_ = ReplacePolicyTags(client, policyName, path)

	return resp

}

func GetPolicyArn(client *iam.Client, name string, path string) (*string, error) {

	var arn *string

	// search for policy using path and name to retrieve the ARN
	policies, err := client.ListPolicies(context.TODO(), &iam.ListPoliciesInput{
		PathPrefix: &path,
	})
	if err != nil {
		fmt.Printf("Could not list policies: %v\n", err)
	} else {
		for _, v := range policies.Policies {
			if *v.Path == path && *v.PolicyName == name {
				arn = v.Arn
			}
		}
	}

	return arn, err
}

func ReplacePolicyTags(client *iam.Client, name string, path string) error {

	var currentTags []string

	tags := DefaultTags()

	arn, err := GetPolicyArn(client, name, path)

	if err != nil {
		// error occurred when trying to retrieve policy arn
		fmt.Printf("GetPolicyArn Error: %v\n", err)
	} else {
		if arn != nil {
			// get the policy and note current tags
			policy, err := client.GetPolicy(context.TODO(), &iam.GetPolicyInput{PolicyArn: arn})
			if err != nil {
				fmt.Printf("Cannot get policy %s: %v\n", *arn, err)
			} else {
				for _, v := range policy.Policy.Tags {
					currentTags = append(currentTags, *v.Key)
				}
			}

			// remove existing tags
			if len(currentTags) != 0 {
				_, err = client.UntagPolicy(context.TODO(), &iam.UntagPolicyInput{
					PolicyArn: arn,
					TagKeys:   currentTags,
				})
				if err != nil {
					fmt.Printf("UntagPolicy Err: %v\n", err)
				}
			}

			// apply the correct tags
			_, err = client.TagPolicy(context.TODO(), &iam.TagPolicyInput{
				PolicyArn: arn,
				Tags:      tags,
			})
			if err != nil {
				fmt.Printf("TagPolicy Err: %v\n", err)
			}
		} else {
			fmt.Printf("Empty ARN Returned: %v\n", err)
		}
	}

	// just return nil for now (the best error handling)
	return nil
}

func ReplaceRoleTags(client *iam.Client, name string, path string) error {

	var currentTags []string
	tags := DefaultTags()

	// get the role and note current tags
	role, err := client.GetRole(context.TODO(), &iam.GetRoleInput{RoleName: &name})
	if err != nil {
		fmt.Printf("Cannot get role %s: %v\n", name, err)
	} else {
		for _, v := range role.Role.Tags {
			currentTags = append(currentTags, *v.Key)
		}
	}

	// remove existing tags
	if len(currentTags) != 0 {
		_, err = client.UntagRole(context.TODO(), &iam.UntagRoleInput{
			RoleName: &name,
			TagKeys:  currentTags,
		})
		if err != nil {
			fmt.Printf("UntagRole Err: %v\n", err)
		}
	}

	// apply the correct tags
	_, err = client.TagRole(context.TODO(), &iam.TagRoleInput{
		RoleName: &name,
		Tags:     tags,
	})
	if err != nil {
		fmt.Printf("TagRole Err: %v\n", err)
	}

	// just return nil for now (the best error handling)
	return nil
}

func AttachIAMPolicy(client *iam.Client, policyArn string, roleName string) {

	var input *iam.AttachRolePolicyInput = &iam.AttachRolePolicyInput{
		PolicyArn: &policyArn,
		RoleName:  &roleName,
	}
	_, err := client.AttachRolePolicy(context.TODO(), input)
	if err != nil {
		// var eae *types.EntityAlreadyExistsException
		// if errors.As(err, &eae) {
		// 	log.Printf("Warning: Role '%s' already exists\n", roleName)
		// } else {
		fmt.Printf("err: %v\n", err)
		// }
	}

}

func DeleteIAMRole(client *iam.Client) {

	name := "Inventory"

	DetachRolePolicies(client, name)

	_, err := client.DeleteRole(context.TODO(), &iam.DeleteRoleInput{RoleName: &name})

	if err != nil {
		fmt.Printf("Error when executing DeleteRole: %v\n", err)
	}

}

func DeleteIAMPolicy(client *iam.Client) {

	name := "Inventory"
	path := "/managed/"

	arn, err := GetPolicyArn(client, name, path)

	if err != nil {
		fmt.Printf("Error retrieving Policy ARN: %v\n", err)
	} else {
		policyVersions, err := client.ListPolicyVersions(context.TODO(), &iam.ListPolicyVersionsInput{PolicyArn: arn})

		if err != nil {
			fmt.Printf("ListPolicyVersions Error: %v\n", err)
		} else {

			// delete all policy versions except the default
			for _, v := range policyVersions.Versions {
				if !v.IsDefaultVersion {
					_, err = client.DeletePolicyVersion(context.TODO(), &iam.DeletePolicyVersionInput{
						PolicyArn: arn,
						VersionId: v.VersionId})
				}

				if err != nil {
					fmt.Printf("DeletePolicyVersion Error: %v\n", err)
				}
			}
		}

		// delete policy (and the default version)
		_, err = client.DeletePolicy(context.TODO(), &iam.DeletePolicyInput{PolicyArn: arn})

		if err != nil {
			fmt.Printf("Error deleting Policy: %v\n", err)
		}
	}

}

func CreateIAMRole(client *iam.Client) {
	roleName := "Inventory"
	path := "/managed/"
	description := "Role for inventory scans"
	assumeRolePolicyDocument := `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "Service": "ec2.amazonaws.com"
            },
            "Action": "sts:AssumeRole"
        }
    ]
}`
	var maxSessionDuration int32 = 3600

	var input *iam.CreateRoleInput = &iam.CreateRoleInput{
		MaxSessionDuration:       &maxSessionDuration,
		AssumeRolePolicyDocument: &assumeRolePolicyDocument,
		Path:                     &path,
		RoleName:                 &roleName,
		Description:              &description,
	}

	_, err := client.CreateRole(context.TODO(), input)
	if err != nil {
		var eae *types.EntityAlreadyExistsException
		if errors.As(err, &eae) {
			log.Printf("Warning: Role '%s' already exists\n", roleName)
		} else {
			fmt.Printf("err: %v\n", err)
		}
	}

	// replace tags associated with the role
	_ = ReplaceRoleTags(client, roleName, path)

}

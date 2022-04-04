package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/fatih/color"
	"golang.org/x/sync/semaphore"

	"github.com/spf13/cobra"
)

func CreatePredefinedIAMRoleCmd(cmd *cobra.Command, args []string) {

	// get parameters from Cobra
	includeAccountIds, _ := cmd.Flags().GetStringSlice("include-account-ids")
	concurrentOps, _ := cmd.Flags().GetInt64("concurrent-operations")

	roleName, _ := cmd.Flags().GetString("role-name")
	bucketName, _ = cmd.Flags().GetString("bucket-name")
	bucketRoleArn, _ := cmd.Flags().GetString("bucket-role-arn")

	// need to assume the role for S3 bucket acess
	fmt.Println(bucketName)
	properties, trustPolicy, inlinePolicy := DownloadRoleDocuments(bucketName, bucketRoleArn, roleName)

	_ = inlinePolicy
	path := properties.Path
	roleDescription := properties.Description
	maxSessionDuration := properties.SessionDuration
	managedPolicies := properties.ManagedPolicies

	//var targetAccounts []orgtypes.Account
	var waitGroup sync.WaitGroup
	sem := semaphore.NewWeighted(concurrentOps)
	ctx := context.TODO()
	startTime := time.Now()

	targetAccounts := make(map[string]string)

	// get list of org accounts
	color.Set(color.FgWhite)
	fmt.Printf("Obtaining a list of Organizational Accounts: ")
	accounts, err := OrgAccountList(includeAccountIds)
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

	for id, creds := range assumedRoles {

		waitGroup.Add(1)

		go func(id string, creds *ststypes.Credentials) {

			color.Set(color.FgWhite)
			fmt.Printf(" Account %s (%s): Creating the Role named '%s'\n", targetAccounts[id], id, roleName)
			sem.Acquire(ctx, 1)
			defer sem.Release(1)
			defer waitGroup.Done()

			cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(*creds.AccessKeyId, *creds.SecretAccessKey, *creds.SessionToken)), config.WithRegion("eu-west-1"))
			if err != nil {
				log.Fatalf("unable to load SDK config, %v", err)
			}

			assumedClient := iam.NewFromConfig(cfg)

			CreateIAMRole(assumedClient, id, roleName, path, trustPolicy, roleDescription, maxSessionDuration)

			AttachIAMPolicy(assumedClient, managedPolicies, roleName)

			color.Set(color.FgGreen)
			fmt.Printf(" Account %s (%s): Role creation complete\n", targetAccounts[id], id)
			color.Unset()
		}(id, creds)
	}

	waitGroup.Wait()

	color.Set(color.FgCyan)
	fmt.Printf("\nTook %f seconds to complete Role creation.\n", time.Since(startTime).Seconds())
	color.Unset()
}

func CreateIAMPolicy(client *iam.Client, id string, policyName string, policyDocument string, path string, description string) (*iam.CreatePolicyOutput, error) {

	var input *iam.CreatePolicyInput = &iam.CreatePolicyInput{
		PolicyName:     &policyName,
		Path:           &path,
		Description:    &description,
		PolicyDocument: &policyDocument,
	}

	resp, err := client.CreatePolicy(context.TODO(), input)

	// in the case of error
	if err != nil {
		var eae *types.EntityAlreadyExistsException

		// handle
		if errors.As(err, &eae) {

			color.Set(color.FgYellow)
			fmt.Printf(" Account %s: (WARN) Policy '%s' already exists\n", id, policyName)

			// Get existing policy ARN
			managedPolicies, err := client.ListPolicies(context.TODO(), &iam.ListPoliciesInput{
				PathPrefix: &path,
			})
			if err != nil {
				//fmt.Printf("Could not list policies: %v\n", err)
				return nil, err
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
				//fmt.Printf("Cannot get policy %s: %v\n", *policyArn, err)
				return nil, err
			}
			policyVersion := policy.Policy.DefaultVersionId

			policyContent, err := client.GetPolicyVersion(context.TODO(), &iam.GetPolicyVersionInput{
				PolicyArn: policyArn,
				VersionId: policyVersion,
			})

			// in the case of an error return it to the calling routine for handling
			if err != nil {
				return nil, err
			}

			currentPolicy, err := url.QueryUnescape(*policyContent.PolicyVersion.Document)

			// in the case of an error return it to the calling routine for handling
			if err != nil {
				return nil, err
			}

			if currentPolicy != policyDocument {
				_, err := client.CreatePolicyVersion(context.TODO(), &iam.CreatePolicyVersionInput{
					PolicyArn:      policyArn,
					PolicyDocument: &policyDocument,
					SetAsDefault:   true,
				})

				// in the case of an error return it to the calling routine for handling
				if err != nil {
					return nil, err
				}

			}

			// Generate policy response
			resp = &iam.CreatePolicyOutput{
				Policy: &types.Policy{
					Arn: policyArn,
				},
			}

		} else {
			// return the error
			return nil, err
		}
	}

	// ensure tags on the policy are replaced with those provided
	err = ReplacePolicyTags(client, policyName, path)

	// if err isn't nil then display the error
	// if err != nil {
	// 	fmt.PrintLn("An error occurred when executing the ReplacePolicyTags function.")
	// 	fmt.Println("The errror was: %v", err)
	// }

	// return response with no errors
	return resp, err

}

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

func CheckRoleExists(client *iam.Client, name string) (bool, error) {

	_, err := client.GetRole(context.TODO(), &iam.GetRoleInput{RoleName: &name})
	if err == nil {
		return true, err
	} else {
		return false, err
	}

}

func DetachRolePolicies(client *iam.Client, name string) error {

	attachedPolicies, err := client.ListAttachedRolePolicies(context.TODO(), &iam.ListAttachedRolePoliciesInput{RoleName: &name})

	// if an error occurred with the last invocation then return the error
	if err != nil {
		return err
	} else {
		for _, v := range attachedPolicies.AttachedPolicies {
			client.DetachRolePolicy(context.TODO(), &iam.DetachRolePolicyInput{
				PolicyArn: v.PolicyArn,
				RoleName:  &name,
			})
		}
	}

	// return err; this should be nil at this point
	return err
}

func DeleteIAMRoleCmd(cmd *cobra.Command, args []string) {

	// get parameters from Cobra
	includeAccountIds, _ := cmd.Flags().GetStringSlice("include-account-ids")
	path, _ := cmd.Flags().GetString("path")
	concurrentOps, _ := cmd.Flags().GetInt64("concurrent-operations")
	roleName, _ := cmd.Flags().GetString("role-name")
	policyName, _ := cmd.Flags().GetString("policy-name")

	if policyName == "" {
		policyName = roleName
	}

	if roleName == "" {
		fmt.Println("No Role Name was specified.")
	} else {
		var waitGroup sync.WaitGroup
		sem := semaphore.NewWeighted(concurrentOps)
		ctx := context.TODO()
		startTime := time.Now()

		targetAccounts := make(map[string]string)

		// get list of org accounts
		color.Set(color.FgWhite)
		fmt.Printf("Obtaining a list of Organizational Accounts: ")
		accounts, err := OrgAccountList(includeAccountIds)
		if err != nil {
			color.Red("Failed")
			color.Yellow("  Error: %v", err)
			os.Exit(1)
		} else {
			for _, v := range accounts {
				fmt.Println(*v.Name)
				//targetAccounts = append(targetAccounts, v)
				targetAccounts[*v.Id] = *v.Name
			}
			color.Green("Done")
		}

		// assume roles in org accounts
		assumedRoles := AssumeRoleMultipleAccounts(targetAccounts)

		for id, creds := range assumedRoles {

			waitGroup.Add(1)

			go func(id string, creds *ststypes.Credentials) {

				fmt.Printf(" Account %s (%s): Deleting the Role named '%s'\n", id, roleName)
				sem.Acquire(ctx, 1)
				defer sem.Release(1)
				defer waitGroup.Done()

				cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(*creds.AccessKeyId, *creds.SecretAccessKey, *creds.SessionToken)), config.WithRegion("eu-west-1"))
				if err != nil {
					log.Fatalf("unable to load SDK config, %v", err)
				}

				assumedClient := iam.NewFromConfig(cfg)

				roleExists, err := CheckRoleExists(assumedClient, roleName)

				if roleExists {
					DeleteIAMRole(assumedClient, roleName)
					DeleteIAMPolicy(assumedClient, policyName, path)
					color.Set(color.FgGreen)
					fmt.Printf(" Account %s (%s): Role deletion complete\n", id)
					color.Unset()
				} else {
					color.Set(color.FgYellow)
					fmt.Printf(" Account %s (%s): (WARN) The Role named '%s' was not found.\n", id, roleName)
					color.Unset()
				}
			}(id, creds)
		}

		waitGroup.Wait()

		color.Set(color.FgCyan)
		fmt.Printf("\nTook %f seconds to complete deletion.\n", time.Since(startTime).Seconds())
		color.Unset()
	}
}

func DeletePredefinedIAMRoleCmd(cmd *cobra.Command, args []string) {
	// Todo: This needs modifying to delete a predefined IAM role

	// get parameters from Cobra
	includeAccountIds, _ := cmd.Flags().GetStringSlice("include-account-ids")
	path, _ := cmd.Flags().GetString("path")
	concurrentOps, _ := cmd.Flags().GetInt64("concurrent-operations")
	roleName, _ := cmd.Flags().GetString("role-name")
	policyName, _ := cmd.Flags().GetString("policy-name")

	if policyName == "" {
		policyName = roleName
	}

	if roleName == "" {
		fmt.Println("No Role Name was specified.")
	} else {
		var waitGroup sync.WaitGroup
		sem := semaphore.NewWeighted(concurrentOps)
		ctx := context.TODO()
		startTime := time.Now()

		targetAccounts := make(map[string]string)

		// get list of org accounts
		color.Set(color.FgWhite)
		fmt.Printf("Obtaining a list of Organizational Accounts: ")
		accounts, err := OrgAccountList(includeAccountIds)
		if err != nil {
			color.Red("Failed")
			color.Yellow("  Error: %v", err)
			os.Exit(1)
		} else {
			for _, v := range accounts {
				fmt.Println(*v.Name)
				//targetAccounts = append(targetAccounts, v)
				targetAccounts[*v.Id] = *v.Name
			}
			color.Green("Done")
		}

		// assume roles in org accounts
		assumedRoles := AssumeRoleMultipleAccounts(targetAccounts)

		for id, creds := range assumedRoles {

			waitGroup.Add(1)

			go func(id string, creds *ststypes.Credentials) {

				fmt.Printf(" Account %s (%s): Deleting the Role named '%s'\n", id, roleName)
				sem.Acquire(ctx, 1)
				defer sem.Release(1)
				defer waitGroup.Done()

				cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(*creds.AccessKeyId, *creds.SecretAccessKey, *creds.SessionToken)), config.WithRegion("eu-west-1"))
				if err != nil {
					log.Fatalf("unable to load SDK config, %v", err)
				}

				assumedClient := iam.NewFromConfig(cfg)

				roleExists, err := CheckRoleExists(assumedClient, roleName)

				if roleExists {
					DeleteIAMRole(assumedClient, roleName)
					DeleteIAMPolicy(assumedClient, policyName, path)
					color.Set(color.FgGreen)
					fmt.Printf(" Account %s (%s): Role deletion complete\n", id)
					color.Unset()
				} else {
					color.Set(color.FgYellow)
					fmt.Printf(" Account %s (%s): (WARN) The Role named '%s' was not found.\n", id, roleName)
					color.Unset()
				}
			}(id, creds)
		}

		waitGroup.Wait()

		color.Set(color.FgCyan)
		fmt.Printf("\nTook %f seconds to complete deletion.\n", time.Since(startTime).Seconds())
		color.Unset()
	}
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
		return err
	} else {
		if arn != nil {
			// get the policy and note current tags
			policy, err := client.GetPolicy(context.TODO(), &iam.GetPolicyInput{PolicyArn: arn})
			if err != nil {
				return err
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
					return err
				}
			}

			// apply the correct tags
			_, err = client.TagPolicy(context.TODO(), &iam.TagPolicyInput{
				PolicyArn: arn,
				Tags:      tags,
			})
			if err != nil {
				return err
			}
		}
		// else {
		// 	fmt.Printf("Empty ARN Returned: %v\n", err)
		// }
	}

	// if we reach here then no error occurred so just return nil
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

func AttachIAMPolicy(client *iam.Client, policyArn []string, roleName string) {

	for _, v := range policyArn {

		var input *iam.AttachRolePolicyInput = &iam.AttachRolePolicyInput{
			PolicyArn: &v,
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

}

func DeleteIAMRole(client *iam.Client, roleName string) bool {

	// try to detach policies
	err := DetachRolePolicies(client, roleName)

	// if the detach of policies succeeded then...
	if err == nil {
		_, err = client.DeleteRole(context.TODO(), &iam.DeleteRoleInput{RoleName: &roleName})

		if err != nil {
			fmt.Printf("Error when executing DeleteRole: %v\n", err)
			return false
		} else {
			return true
		}
	} else {
		return false
	}

}

func DeleteIAMPolicy(client *iam.Client, policyName string, path string) {

	// get the ARN for the policy
	arn, err := GetPolicyArn(client, policyName, path)

	if err != nil {
		fmt.Printf("Error retrieving Policy ARN: %v\n", err)
	} else {
		// get policy versions
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

func CreateIAMRole(client *iam.Client, id string, rolename string, path string, assumeRolePolicyDocument string, description string, maxSessionDuration int32) {

	var input *iam.CreateRoleInput = &iam.CreateRoleInput{
		MaxSessionDuration:       &maxSessionDuration,
		AssumeRolePolicyDocument: &assumeRolePolicyDocument,
		Path:                     &path,
		RoleName:                 &rolename,
		Description:              &description,
	}

	_, err := client.CreateRole(context.TODO(), input)
	if err != nil {
		var eae *types.EntityAlreadyExistsException
		if errors.As(err, &eae) {
			color.Set(color.FgYellow)
			fmt.Printf(" Account %s: (WARN) Role '%s' already exists\n", id, rolename)
			color.Unset()
		} else {
			fmt.Printf("err: %v\n", err)
		}
	}

	// replace tags associated with the role
	_ = ReplaceRoleTags(client, rolename, path)

}

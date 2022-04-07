package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/dfds/ce-cli/util"
	"github.com/fatih/color"
	"golang.org/x/sync/semaphore"

	"github.com/spf13/cobra"
)

const DEFAULT_INLINE_POLICY_NAME string = "inlinePolicy"

func CreateIAMOIDCProviderCmd(cmd *cobra.Command, args []string) {

	// get parameters from cobra
	url, _ := cmd.Flags().GetString("url")
	includeAccountIds, _ := cmd.Flags().GetStringSlice("include-account-ids")
	concurrentOps, _ := cmd.Flags().GetInt64("concurrent-operations")

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

		go func(id string, creds *ststypes.Credentials, url string) {

			color.Set(color.FgWhite)
			fmt.Printf(" Account %s (%s): Creating an IAM OpenID Connect Provider.\n", targetAccounts[id], id)
			sem.Acquire(ctx, 1)
			defer sem.Release(1)
			defer waitGroup.Done()

			cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(*creds.AccessKeyId, *creds.SecretAccessKey, *creds.SessionToken)), config.WithRegion("eu-west-1"))
			if err != nil {
				log.Fatalf("unable to load SDK config, %v", err)
			}

			// get a new client used the config we just generated
			assumedClient := iam.NewFromConfig(cfg)

			// try to create the OpenID Connect Provider
			err = CreateOpenIDConnectProvider(assumedClient, url)

			if err != nil {
				// handle the error
				var eae *types.EntityAlreadyExistsException
				if errors.As(err, &eae) {
					color.Set(color.FgYellow)
					fmt.Printf(" Account %s (%s): (WARN) The OpenID Connect Provider already exists\n", targetAccounts[id], id)
					color.Set(color.FgWhite)
				} else {
					if err != nil {
						color.Set(color.FgYellow)
						fmt.Printf(" Account %s (%s): (ERR) An error occurred when trying to create the OpenID Connect Provider.\n", targetAccounts[id], id)
						fmt.Printf(" Account %s (%s): (ERR) The error was: %v\n", targetAccounts[id], id, err)
						color.Set(color.FgWhite)
					}
				}
			} else {
				color.Set(color.FgGreen)
				fmt.Printf(" Account %s (%s): IAM OpenID Connect Provider creation complete\n", targetAccounts[id], id)
				color.Unset()
			}
		}(id, creds, url)
	}

	waitGroup.Wait()

	color.Set(color.FgCyan)
	fmt.Printf("\nTook %f seconds to complete IAM OpenID Connect Provider creation.\n", time.Since(startTime).Seconds())
	color.Unset()
}

func CreatePredefinedIAMRoleCmd(cmd *cobra.Command, args []string) {

	// get parameters from Cobra
	includeAccountIds, _ := cmd.Flags().GetStringSlice("include-account-ids")
	concurrentOps, _ := cmd.Flags().GetInt64("concurrent-operations")

	roleName, _ := cmd.Flags().GetString("role-name")
	bucketName, _ = cmd.Flags().GetString("bucket-name")
	bucketRoleArn, _ := cmd.Flags().GetString("bucket-role-arn")

	// need to assume the role for S3 bucket acess
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

			// create the role
			CreateIAMRole(assumedClient, targetAccounts[id], id, roleName, path, trustPolicy, roleDescription, maxSessionDuration)

			// create custom inline policy
			CreateIAMRoleInlinePolicy(assumedClient, roleName, inlinePolicy, DEFAULT_INLINE_POLICY_NAME)

			// attach managed policies
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

	// attempt to get a list of attached policies for the specified role name
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
	concurrentOps, _ := cmd.Flags().GetInt64("concurrent-operations")
	roleName, _ := cmd.Flags().GetString("role-name")

	if roleName == "" {
		fmt.Println("No Role Name was specified.")
	} else {

		// goroutine management
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

				fmt.Printf(" Account %s (%s): Deleting the Role named '%s'\n", targetAccounts[id], id, roleName)
				sem.Acquire(ctx, 1)
				defer sem.Release(1)
				defer waitGroup.Done()

				cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(*creds.AccessKeyId, *creds.SecretAccessKey, *creds.SessionToken)), config.WithRegion("eu-west-1"))
				if err != nil {
					log.Fatalf("unable to load SDK config, %v", err)
				}

				assumedClient := iam.NewFromConfig(cfg)

				roleExists, _ := CheckRoleExists(assumedClient, roleName)

				if roleExists {

					DeleteIAMRole(assumedClient, roleName)
					color.Set(color.FgGreen)
					fmt.Printf(" Account %s (%s): Role deletion complete\n", targetAccounts[id], id)
					color.Unset()
				} else {
					color.Set(color.FgYellow)
					fmt.Printf(" Account %s (%s): (WARN) The Role named '%s' was not found.\n", targetAccounts[id], id, roleName)
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

func DeleteIAMRoleInlinePolicy(client *iam.Client, roleName string, policyName string) error {

	// attempt to delete the specified inline policy from the specified role
	_, err := client.DeleteRolePolicy(context.TODO(), &iam.DeleteRolePolicyInput{RoleName: &roleName, PolicyName: &policyName})
	return err

}
func DeleteIAMRole(client *iam.Client, roleName string) bool {

	// try to detach managed policies
	err := DetachRolePolicies(client, roleName)

	// if the detach of policies succeeded then...
	if err == nil {

		// attempt to delete the inline policy
		err = DeleteIAMRoleInlinePolicy(client, roleName, DEFAULT_INLINE_POLICY_NAME)

		// don't report an error if it was just due to the policy not being found, but do report other errors
		var epnf *types.NoSuchEntityException
		if err != nil {
			if !errors.As(err, &epnf) {
				color.Set(color.FgYellow)
				fmt.Printf(" An error occurred whilst trying to delete the inline policy named %s.\n", DEFAULT_INLINE_POLICY_NAME)
				fmt.Printf(" The error was: %v\n", err)
				return false
			}
		}

		// delete the role
		_, err = client.DeleteRole(context.TODO(), &iam.DeleteRoleInput{RoleName: &roleName})

		// return true or false depending on if the final deletion completed succesfully
		if err != nil {
			fmt.Printf("Error when executing DeleteRole: %v\n", err)
			return false
		} else {
			return true
		}
	}

	// if we reach here then something went wrong so return false
	return false

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

func CreateIAMRoleInlinePolicy(client *iam.Client, roleName string, inlinePolicy string, policyName string) {
	// Note that this function will replace the existing inlinePolicy with a new one if the same policyName is used.
	// The function can therefore be used for both creating an initial policy and updating the existing policy

	// define input for the policy put request
	var input *iam.PutRolePolicyInput = &iam.PutRolePolicyInput{
		PolicyDocument: &inlinePolicy,
		RoleName:       &roleName,
		PolicyName:     &policyName,
	}

	// put the inline policy in place
	_, err := client.PutRolePolicy(context.TODO(), input)

	if err != nil {
		color.Set(color.FgYellow)
		fmt.Println(" There was a problem whilst trying to create the inline policy.")
		fmt.Printf(" The error was: %v\n", err)
		color.Unset()
	}
}

func CreateOpenIDConnectProvider(client *iam.Client, url string) error {

	// split the provided URL into component parts
	urlParts := strings.Split(url, "/")

	// get the host component
	server := urlParts[2]

	// set default ssl port
	var port uint = 443

	// get the thumb print for the provider
	thumbPrintList := util.GetCertificateSHAThumbprint(&server, &port)

	// build clientIDList with standard audient
	clientIDList := make([]string, 1)
	clientIDList[0] = "sts.amazonaws.com"

	// get the default tags
	tags := DefaultTags()

	// build input for creation
	var input *iam.CreateOpenIDConnectProviderInput = &iam.CreateOpenIDConnectProviderInput{
		ThumbprintList: thumbPrintList,
		Url:            &url,
		ClientIDList:   clientIDList,
		Tags:           tags,
	}

	// create provider
	_, err := client.CreateOpenIDConnectProvider(context.TODO(), input)

	// return content of err
	return err
}

func CreateIAMRole(client *iam.Client, accountName string, accountId string, rolename string, path string, assumeRolePolicyDocument string, description string, maxSessionDuration int32) {

	// define input for the role creation
	var input *iam.CreateRoleInput = &iam.CreateRoleInput{
		MaxSessionDuration:       &maxSessionDuration,
		AssumeRolePolicyDocument: &assumeRolePolicyDocument,
		Path:                     &path,
		RoleName:                 &rolename,
		Description:              &description,
	}

	// try to create the required role
	_, err := client.CreateRole(context.TODO(), input)

	// in the case of an error
	if err != nil {
		var eae *types.EntityAlreadyExistsException
		if errors.As(err, &eae) {
			color.Set(color.FgYellow)
			fmt.Printf(" Account %s (%s): (WARN) Role '%s' already exists\n", accountName, accountId, rolename)
			color.Unset()

			// if the role already existed then at least ensure the AssumeRolePolicyDocument is updated
			_, err = client.UpdateAssumeRolePolicy(context.TODO(), &iam.UpdateAssumeRolePolicyInput{PolicyDocument: &assumeRolePolicyDocument, RoleName: &rolename})

			// display errors if any occurred
			if err != nil {
				color.Set(color.FgYellow)
				fmt.Printf(" Account %s (%s): (ERR) An error occurred when trying to update the AssumeRolePolicy for the Role.\n", accountName, accountId)
				fmt.Printf(" Account %s (%s): (ERR) The error was: %v\n", accountName, accountId, err)
				color.Unset()
			}
		} else {
			color.Set(color.FgYellow)
			fmt.Printf(" Account %s (%s): (WARN) An error occurred whilst trying to create the new Role.\n", accountName, accountId)
			fmt.Printf("err: %v\n", err)
		}
	}

	// replace tags associated with the role
	_ = ReplaceRoleTags(client, rolename, path)

}

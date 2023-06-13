# Cloud Engineering CLI

Internal CLI tool used by DFDS' Cloud Engineering teeam used for mass-manage various cloud resources, e.g. across all AWS accounts in our AWS Organization.

For AWS, the idea more specifically is to have an easy-to-use way of deploying resources, that requires very high privileges - i.e. the ability to assume the organization role in each AWS account.

With this, it's easy to provision IAM roles with lesser privileges that can then be used for various use cases, like running performing inventory of AWS accounts (more or less only `Get*`, `List*`, `Describe*` permissions).

## Getting help

Use the `--help` argument to get help for the current command scope, e.g. `ce aws --help`, `ce aws create-predefined-iam-role --help`.

## AWS features

All the AWS commands currently the AWS Organization, and attempts to assume the organization role in each account, thus requires admin rights to the billing account. It uses the AWS SDK's default credential search order.

The suggested way to authenticate, is to login using `go-aws-sso -p admin` and assume the admin role. Then set the `AWS_PROFILE` environment variable with `export AWS_PROFILE=admin`.

### AWS commands

| Command                           | Description                                                                                   |
| --------------------------------- | --------------------------------------------------------------------------------------------- |
| `create-predefined-iam-role`      | Create a pre-defined IAM role, based on policies read from the S3 bucket.                     |
| `delete-iam-role`                 | Delete the specified IAM role.                                                                |
| `create-oidc-provider`            | Create an IAM Open ID Connect Provider using the endpoint e.g. from EKS cluster.              |
| `update-oidc-provider-thumbprint` | Updates the thumbprint associated with an IAM Open ID Connect Provider.                       |
| `delete-oidc-provider`            | Delete an IAM Open ID Connect Provider.                                                       |
| `list-org-accounts`               | Returns all AWS accounts in the Organization, optionally filtered by `--include-account-ids`. |

At least the `create-predefined-iam-role` command requires a backend bucket is specified (see "Backend S3 bucket"), and a role ARN to assume in order to read from it.

Example:

```bash
ce aws create-predefined-iam-role --bucket-name "${BACKEND_S3_BUCKET}" --bucket-role-arn "${BACKEND_IAM_ROLE_ARN}" --role-name "inventory"
```

Substitute `${BACKEND_S3_BUCKET}` and `${BACKEND_IAM_ROLE_ARN}` (typically in the *security* account).

### Common AWS arguments

| Argument                  | Short | Description                                                                                              |
| ------------------------- | ----- | -------------------------------------------------------------------------------------------------------- |
| `--include-account-ids`   | `-i`  | Filter the AWS Organization account IDs.<br>If omitted, *all* accounts in the Organization are returned. |
| `--exclude-account-ids`   | `-e`  | Specifically exclude the specified account IDs.                                                          |
| `--path`                  | `-p`  | The path (prefix) for resource names where applicable, e.g. IAM roles.                                   |
| `--concurrent-operations` | `-c`  | Maximum number of concurrent operations for parallel operations.                                         |

### Configuration File

If you do not wish to have to provide the bucket-name and bucket-role-arn parameters with each command execution then you can create a configuration file.  Once the file is defined and in place you no longer need to specify the values.

To do this create a file named `config.yaml` inside of a folder named `.ce-cli` in your home directory.  An example of what the file should contain is displayed below.

```
bucket-name: ${BACKEND_S3_BUCKET}
bucket-role-arn: ${BACKEND_IAM_ROLE_ARN}
```

### Exclude certain accounts based on scope

In certain cases, it's not desireable to deploy resource of a particular type to certain accounts. In DFDS' case, OIDC providers in a few AWS account for our production Kubernetes clusters are managed by an EKS pipeline. In order to use [*IAM Roles for Service Accounts*](https://docs.aws.amazon.com/emr/latest/EMR-on-EKS-DevelopmentGuide/setting-up-enable-IAM.html) to assume roles in *other* AWS accounts, we need to deploy OIDC providers here as well, referencing the Kubernetes OIDC endpoint - without interfering with those managed by the pipeline (or other tools).

You *could* pass those account IDs through `--exclude-account-ids`, but forgetting so, might cause the resource to be overwritten anyway. Instead, some commands look in the backend S3 bucket, for any accounts that should *always* be excluded for certain operations (aka. *scopes*).

The scopes, their associated, excluded, account ids are stored in JSON format, and include a "Common" scope, that is applied for all commands using the file. Example:

```json
{
    "Scopes": {
        "Common": [
            "111111111111"
        ],
        "OidcProvider": [
            "222222222222",
            "333333333333"
        ]
    }
}
```

Currently the following account exclusion files and scopes exist:

| File                             | Scope          | Commands                                       |
| -------------------------------- | -------------- | ---------------------------------------------- |
| `aws/iam/excludeAccountIds.json` | `OidcProvider` | `create-oidc-provider`, `delete-oidc-provider` |
| `aws/org/excludeAccountIds.json` | `ListAccounts` | `list-accounts`                                |

It's far from perfect and could certainly be more sophisticated. One limitation is discussed in issue #14.

## Backend S3 bucket

All policy documents for the predefined AWS IAM roles are read from the specified S3 bucket. In the future, other templates and configuration might be read from here.

We have provisioned this bucket in our *security* account. An IAM role is needed to access the bucket, which can be assumed by an admin of the AWS billing account.

### Backend bucket IAM policies

**Permission policy:**

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "VisualEditor0",
            "Effect": "Allow",
            "Action": [
                "s3:GetObject",
                "s3:ListBucket"
            ],
            "Resource": [
                "arn:aws:s3:::${BACKEND_S3_BUCKET}/*",
                "arn:aws:s3:::${BACKEND_S3_BUCKET}"
            ]
        }
    ]
}
```

Substitute `${BACKEND_S3_BUCKET}`.

**Trust policy:**

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::${BILLING_ACCOUNT_ID}:root"
            },
            "Action": "sts:AssumeRole",
            "Condition": {}
        }
    ]
}
```

Substitute `${BILLING_ACCOUNT_ID}`.

### Backend bucket structure

The expected structure for AWS IAM role files in the bucket is:

`aws/iam/${PREDEFINED_ROLE_NAME}-role/`.

The following files are needed:

| File              | Description                                                                          |
| ----------------- | ------------------------------------------------------------------------------------ |
| `policy.json`     | The inline permission policy document to attach                                      |
| `properties.json` | Various properties for the role, including any managed permission policies to attach |
| `trust.json`      | The role trust policy document                                                       |

#### `properties.json`

The `properties.json` file contains the following properties:

| Property          | Description                       |
| ----------------- | --------------------------------- |
| `description`     | Role description                  |
| `sessionDuration` | Maximum role session duration     |
| `path`            | Role name prefix (aka. path)      |
| `managedPolicies` | List of IAM policy ARNs to attach |

Example file for the `inventory` predefined IAM role:

```json
{
    "description" : "Inventory role description.",
    "sessionDuration" : 28800,
    "path" : "/managed/",
    "managedPolicies" : [
        "arn:aws:iam::aws:policy/job-function/ViewOnlyAccess"
    ]
}
```

# Provision S3 bucket for support files

## Apply

```bash
terraform init
terraform apply -var='assume_role_account_id=ACCOUNT_ID' -var='bucket_name=BUCKET_NAME'
```

Substitute `ACCOUNT_ID` and `BUCKET_NAME` as needed.

## Destroy

```bash
terraform destroy -var='assume_role_account_id=ACCOUNT_ID' -var='bucket_name=BUCKET_NAME'
```

Substitute `ACCOUNT_ID` and `BUCKET_NAME` as needed.
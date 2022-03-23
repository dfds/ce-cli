provider "aws" {
  region = var.aws_region
  assume_role {
    role_arn = "arn:aws:iam::${var.assume_role_account_id}:role/OrgRole"
  }
}

module "bucket" {
  source    = "git::https://github.com/dfds/infrastructure-modules.git//_sub/storage/s3-bucket?ref=0.6.5"
  s3_bucket = var.bucket_name
}

module "iam_inventory_role_policy" {
  source  = "git::https://github.com/dfds/infrastructure-modules.git//_sub/storage/s3-bucket-object?ref=0.6.5"
  bucket  = module.bucket.bucket_name
  key     = "aws/iam/inventory-role/policy.json"
  content = data.template_file.iam_inventory_role_policy.rendered
}

module "iam_inventory_role_trust" {
  source  = "git::https://github.com/dfds/infrastructure-modules.git//_sub/storage/s3-bucket-object?ref=0.6.5"
  bucket  = module.bucket.bucket_name
  key     = "aws/iam/inventory-role/trust.json"
  content = data.template_file.iam_inventory_role_trust.rendered
}
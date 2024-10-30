terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
    }
  }

}

provider "aws" {
  region  = "us-west-2"
}


##------------------------------------------------------------------------------
## Local
##------------------------------------------------------------------------------
locals {
  name        = "athena-smurf"
  environment = "test"
  label_order = ["name", "environment"]
}

##------------------------------------------------------------------------------
## AWS S3
##------------------------------------------------------------------------------
module "s3_bucket" {
  source        = "clouddrove/s3/aws"
  version       = "2.0.0"
  name          = format("%s-bucket-test", local.name)
  versioning    = true
  acl           = "private"
  force_destroy = true

}
 terraform {
  required_providers{
    aws={
        source = "hashicorp/aws"
        version = "4.33.0"
    }
  }
}

provider "aws"{
    #configuration options
    region = "us-east-1"
}
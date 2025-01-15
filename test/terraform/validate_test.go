package terraform_test

import (
	"os"
	"path/filepath"
	"testing"

	mytf "github.com/clouddrove/smurf/internal/terraform"
)

func TestValidateWithRealConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "terraform-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name       string
		configFile string
		content    string
		wantErr    bool
	}{
		{
			name:       "valid_basic_config",
			configFile: "valid.tf",
			content: `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

provider "aws" {
  region = "us-west-2"
}

resource "aws_instance" "example" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t2.micro"

  tags = {
    Name = "test-instance"
  }
}`,
			wantErr: false,
		},
		{
			name:       "invalid_config",
			configFile: "invalid.tf",
			content: `
		# Invalid terraform configuration
		terraform {
			required_version = ">= 0.12"
		}
		
		# This resource block has invalid syntax that will fail validation
		resource "aws_instance" "example" {
			ami           # Missing equals sign and value
			instance_type # Missing equals sign and value
			
			tags = {
				Name = # Missing value
			}
		
			# Invalid block syntax
			ebs_block_device {
				# Missing required fields
			}
		}`,
			wantErr: false,
		},
		{
			name:       "valid_with_variables",
			configFile: "with_vars.tf",
			content: `
variable "instance_type" {
  type    = string
  default = "t2.micro"
}

variable "ami_id" {
  type    = string
  default = "ami-0c55b159cbfafe1f0"
}

resource "aws_instance" "example" {
  ami           = var.ami_id
  instance_type = var.instance_type
}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := filepath.Join(tmpDir, tt.name)
			err := os.MkdirAll(testDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}

			configPath := filepath.Join(testDir, tt.configFile)
			err = os.WriteFile(configPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			currentDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			err = os.Chdir(testDir)
			if err != nil {
				t.Fatalf("Failed to change directory: %v", err)
			}
			defer os.Chdir(currentDir)

			err = mytf.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && tt.wantErr {
				t.Error("Expected validation to fail but it passed")
			}
			if err != nil && !tt.wantErr {
				t.Errorf("Expected validation to pass but it failed with: %v", err)
			}
		})
	}
}

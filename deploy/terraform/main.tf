terraform {
  required_providers {
    huaweicloud = {
      source  = "huaweicloud/huaweicloud"
      version = "~> 1.60"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.24"
    }
  }
}

# Variables
variable "region" {
  default = "sa-brazil-1"
}

variable "obs_bucket_name" {
  default = "finops-focus-data"
}

variable "postgres_instance_name" {
  default = "finops-postgres"
}

variable "redis_instance_name" {
  default = "finops-redis"
}

# Huawei OBS Bucket
resource "huaweicloud_obs_bucket" "focus_data" {
  bucket        = var.obs_bucket_name
  storage_class = "STANDARD"
  acl           = "private"

  versioning = true

  lifecycle_rule {
    name    = "archive_old"
    enabled = true

    transition {
      days          = 90
      storage_class = "WARM"
    }

    transition {
      days          = 365
      storage_class = "COLD"
    }
  }

  tags = {
    Environment = "production"
    Project     = "finops"
    ManagedBy   = "terraform"
  }
}

# IAM User for OBS Access
resource "huaweicloud_identity_user" "finops_ingestion" {
  name     = "finops-ingestion"
  pwd_mode = "noCare"
}

resource "huaweicloud_identity_access_key" "finops_ingestion" {
  user_id = huaweicloud_identity_user.finops_ingestion.id
}

resource "huaweicloud_obs_bucket_policy" "focus_policy" {
  bucket = huaweicloud_obs_bucket.focus_data.bucket
  policy = jsonencode({
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          ID = [huaweicloud_identity_user.finops_ingestion.id]
        }
        Action   = ["obs:object:GetObject", "obs:object:ListBucket"]
        Resource = [
          "${huaweicloud_obs_bucket.focus_data.bucket}",
          "${huaweicloud_obs_bucket.focus_data.bucket}/*"
        ]
      }
    ]
  })
}

# RDS PostgreSQL (simplified - use actual resource types for your region)
# Note: Actual resource names vary by region. Check Huawei Cloud documentation.

# DCS Redis (simplified)
# Note: Actual resource names vary by region.

# Load Balancer
resource "huaweicloud_elb_loadbalancer" "finops_lb" {
  name           = "finops-lb"
  vpc_id         = var.vpc_id  # Must exist
  subnet_id      = var.subnet_id  # Must exist
  bandwidth      = 100
  type           = "External"
  admin_state_up = true
}

# Kubernetes/CCE Cluster (data source - assumes existing)
data "huaweicloud_cce_cluster" "finops" {
  name = "finops-cluster"
}

# Outputs
output "obs_bucket_name" {
  value = huaweicloud_obs_bucket.focus_data.bucket
}

output "obs_endpoint" {
  value = huaweicloud_obs_bucket.focus_data.bucket_domain_name
}

output "ingestion_access_key" {
  value     = huaweicloud_identity_access_key.finops_ingestion.id
  sensitive = true
}

output "ingestion_secret_key" {
  value     = huaweicloud_identity_access_key.finops_ingestion.secret
  sensitive = true
}

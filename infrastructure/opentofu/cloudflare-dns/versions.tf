terraform {
  required_version = ">= 1.12.0, < 1.13.0"

  backend "s3" {
    bucket                      = "homelab-tofu-state"
    key                         = "cloudflare-dns/terraform.tfstate"
    region                      = "auto"
    endpoints                   = { s3 = "https://e57deed2a6fbc6ca1d3570353f246f7a.r2.cloudflarestorage.com" }
    use_path_style              = true
    use_lockfile                = true
    skip_credentials_validation = true
    skip_metadata_api_check     = true
    skip_region_validation      = true
    skip_requesting_account_id  = true
    skip_s3_checksum            = true
  }

  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "5.28.0"
    }
  }
}

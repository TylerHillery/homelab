resource "cloudflare_r2_bucket" "state" {
  account_id = "e57deed2a6fbc6ca1d3570353f246f7a"
  name       = "homelab-tofu-state"

  lifecycle {
    prevent_destroy = true
  }
}

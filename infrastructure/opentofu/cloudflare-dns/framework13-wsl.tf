data "cloudflare_zone" "tylertries" {
  filter = {
    name = "tylertries.com"
  }
}

locals {
  framework13_wsl_tailnet_ip = "100.102.61.98"
}

resource "cloudflare_dns_record" "opencode_framework13_wsl" {
  zone_id = data.cloudflare_zone.tylertries.id
  name    = "opencode.framework13-wsl.homelab.tylertries.com"
  type    = "A"
  content = local.framework13_wsl_tailnet_ip
  ttl     = 300
  proxied = false
  comment = "Tailnet-only OpenCode on Framework WSL; ingress managed separately"
}

resource "cloudflare_dns_record" "hlims_staging_framework13_wsl" {
  zone_id = data.cloudflare_zone.tylertries.id
  name    = "hlims.staging.framework13-wsl.homelab.tylertries.com"
  type    = "A"
  content = local.framework13_wsl_tailnet_ip
  ttl     = 300
  proxied = false
  comment = "Tailnet-only HLIMS staging on Framework WSL; ingress managed separately"
}

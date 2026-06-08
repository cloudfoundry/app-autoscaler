#!/bin/bash
set -euo pipefail

# Patches bbl-template.tf after bbl plan to replace the global HTTPS load
# balancer with a regional TCP network load balancer. This is needed because
# BBL v9's Terraform override mechanism is broken for this use case in TF 1.4+.

TERRAFORM_DIR="${1:?Usage: patch-bbl-template.sh <terraform-dir>}"

cd "${TERRAFORM_DIR}"

if [ ! -f bbl-template.tf ]; then
  echo "ERROR: bbl-template.tf not found in ${TERRAFORM_DIR}"
  exit 1
fi

echo "Patching bbl-template.tf to use regional network LB..."

# Remove the global LB resources and replace with regional equivalents.
# We use a python script for reliable multi-line editing.
python3 - bbl-template.tf <<'PYTHON'
import sys

with open(sys.argv[1], 'r') as f:
    content = f.read()

# Resources to remove entirely (global LB stack)
remove_resources = [
    'resource "google_compute_global_address" "cf-address"',
    'resource "google_compute_global_forwarding_rule" "cf-http-forwarding-rule"',
    'resource "google_compute_global_forwarding_rule" "cf-https-forwarding-rule"',
    'resource "google_compute_target_http_proxy" "cf-http-lb-proxy"',
    'resource "google_compute_target_https_proxy" "cf-https-lb-proxy"',
    'resource "google_compute_ssl_certificate" "cf-cert"',
    'resource "google_compute_url_map" "cf-https-lb-url-map"',
    'resource "google_compute_health_check" "cf-public-health-check"',
    'resource "google_compute_backend_service" "router-lb-backend-service"',
    'resource "google_compute_instance_group" "router-lb-0"',
    'resource "google_compute_instance_group" "router-lb-1"',
    'resource "google_compute_instance_group" "router-lb-2"',
    'resource "google_compute_address" "cf-ws"',
    'resource "google_compute_target_pool" "cf-ws"',
    'resource "google_compute_forwarding_rule" "cf-ws-https"',
    'resource "google_compute_forwarding_rule" "cf-ws-http"',
    'resource "google_dns_record_set" "doppler-dns"',
    'resource "google_dns_record_set" "loggregator-dns"',
    'resource "google_dns_record_set" "wildcard-ws-dns"',
]

# Outputs to remove (we'll re-add them with correct values)
remove_outputs = [
    'output "router_backend_service"',
    'output "router_lb_ip"',
    'output "ws_lb_ip"',
    'output "ws_target_pool"',
]

# Also remove/replace these resources (we'll provide new versions)
replace_resources = [
    'resource "google_compute_firewall" "cf-health-check"',
    'resource "google_compute_firewall" "firewall-cf"',
    'resource "google_dns_record_set" "wildcard-dns"',
]

def remove_block(text, header):
    """Remove a terraform block (resource/output) starting with header."""
    idx = text.find(header)
    if idx == -1:
        return text
    # Find the opening brace
    brace_start = text.find('{', idx)
    if brace_start == -1:
        return text
    # Find matching closing brace
    depth = 0
    i = brace_start
    while i < len(text):
        if text[i] == '{':
            depth += 1
        elif text[i] == '}':
            depth -= 1
            if depth == 0:
                # Remove from start of line containing header to end of block
                line_start = text.rfind('\n', 0, idx)
                if line_start == -1:
                    line_start = 0
                end = i + 1
                # Consume exactly one trailing newline to avoid collapsing blocks
                if end < len(text) and text[end] == '\n':
                    end += 1
                text = text[:line_start + 1] + text[end:]
                break
        i += 1
    return text

for r in remove_resources + remove_outputs + replace_resources:
    content = remove_block(content, r)

# Also remove the ssl_certificate and ssl_certificate_private_key variables
# (only if they're not needed by other resources — check first)
# Actually these may be needed by bbl, leave them.

# Append the regional LB resources
regional_lb = '''
# --- Regional Network LB (replaces global HTTPS LB) ---

resource "google_compute_address" "cf-address" {
  name = "${var.env_id}-cf"
}

resource "google_compute_target_pool" "router-lb-target-pool" {
  name = "${var.env_id}-router-lb"

  health_checks = [
    google_compute_http_health_check.cf-public-health-check.name,
  ]
}

resource "google_compute_forwarding_rule" "cf-http-forwarding-rule" {
  name        = "${var.env_id}-cf-http"
  ip_address  = google_compute_address.cf-address.address
  target      = google_compute_target_pool.router-lb-target-pool.self_link
  port_range  = "80"
  ip_protocol = "TCP"
}

resource "google_compute_forwarding_rule" "cf-https-forwarding-rule" {
  name        = "${var.env_id}-cf-https"
  ip_address  = google_compute_address.cf-address.address
  target      = google_compute_target_pool.router-lb-target-pool.self_link
  port_range  = "443"
  ip_protocol = "TCP"
}

resource "google_compute_firewall" "cf-health-check" {
  name    = "${var.env_id}-cf-health-check"
  network = google_compute_network.bbl-network.name

  allow {
    protocol = "tcp"
    ports    = ["8080", "80"]
  }

  source_ranges = ["130.211.0.0/22", "35.191.0.0/16"]
  target_tags   = [google_compute_target_pool.router-lb-target-pool.name]
}

resource "google_compute_firewall" "firewall-cf" {
  name    = "${var.env_id}-cf-open"
  network = google_compute_network.bbl-network.name

  allow {
    protocol = "tcp"
    ports    = ["80", "443"]
  }

  source_ranges = ["0.0.0.0/0"]

  target_tags = [google_compute_target_pool.router-lb-target-pool.name]
}

resource "google_dns_record_set" "wildcard-dns" {
  name = "*.${google_dns_managed_zone.env_dns_zone.dns_name}"
  type = "A"
  ttl  = 300

  managed_zone = google_dns_managed_zone.env_dns_zone.name

  rrdatas = [google_compute_address.cf-address.address]
}

resource "google_dns_record_set" "wildcard-tcp-dns" {
  name = "*.tcp.${google_dns_managed_zone.env_dns_zone.dns_name}"
  type = "A"
  ttl  = 300

  managed_zone = google_dns_managed_zone.env_dns_zone.name

  rrdatas = [google_compute_address.cf-tcp-router.address]
}

output "router_backend_service" {
  value = ""
}

output "router_target_pool" {
  value = google_compute_target_pool.router-lb-target-pool.name
}

output "router_lb_ip" {
  value = google_compute_address.cf-address.address
}

output "ws_lb_ip" {
  value = ""
}

output "ws_target_pool" {
  value = ""
}
'''

content = content.rstrip() + '\n' + regional_lb

with open(sys.argv[1], 'w') as f:
    f.write(content)

print("Done: bbl-template.tf patched for regional network LB")
PYTHON

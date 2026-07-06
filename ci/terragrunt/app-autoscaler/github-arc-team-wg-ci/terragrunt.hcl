locals {
  config = yamldecode(file("./config.yaml"))
}

remote_state {
  backend = "gcs"
  generate = {
    path      = "backend.tf"
    if_exists = "overwrite"
  }
  config = {
    bucket         = "${local.config.gcs_bucket}"
    prefix         = "github-arc-team-${local.config.team_name}"
    project        = "${local.config.project}"
    location       = "${local.config.region}"
    # use for uniform bucket-level access
    # (https://cloud.google.com/storage/docs/uniform-bucket-level-access)
    enable_bucket_policy_only = true
  }
}

# git for teams
terraform {
  source = local.config.tf_modules.github_arc
}

inputs = {
  project = local.config.project
  region  = local.config.region
  zone    = local.config.zone

  team_name = local.config.team_name

  github_repos = tolist(local.config.github_repos)

  gke_name = local.config.gke_name
  gke_arc_node_pool_disk_size_gb = local.config.gke_arc_node_pool_disk_size_gb
  gke_arc_node_pool_machine_type = local.config.gke_arc_node_pool_machine_type
  gke_arc_node_pool_count = local.config.gke_arc_node_pool_count
  gke_arc_node_pool_autoscaling_max = local.config.gke_arc_node_pool_autoscaling_max
  gke_arc_node_pool_ssd_count = local.config.gke_arc_node_pool_ssd_count
  gke_arc_runner_storage_type = local.config.gke_arc_runner_storage_type

  arc_webhook_server_name = "${local.config.gke_name}-arc"
  arc_webhook_server_domain = "${local.config.arc_webhook_server_domain}"
  arc_webhook_server_token_name =  "${local.config.gke_name}-arc-webhook-server-token"


}


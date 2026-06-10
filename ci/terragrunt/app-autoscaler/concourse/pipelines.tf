resource "concourse_pipeline" "api_tester" {
  team_name     = "app-autoscaler"
  pipeline_name = "api-tester"

  is_exposed = false
  is_paused  = false

  pipeline_config        = file("pipelines/api-tester/pipeline.yml")
  pipeline_config_format = "yaml"
}

terraform {
  required_providers {
    concourse = {
      source = "terraform-provider-concourse/concourse"
    }
  }

}


provider "concourse" {
  target = var.fly_target
}
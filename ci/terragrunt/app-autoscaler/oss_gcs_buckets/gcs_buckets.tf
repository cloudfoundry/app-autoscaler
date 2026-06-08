resource "google_service_account" "app_autoscaler_oss_blobstore_uploader" {
  #id is shortened due to lenght limitations
  account_id   = "app-autoscaler-blobstore-oss-u"
  display_name = "app-autoscaler-blobstore-oss-uploader"
  description  = "Can upload to the app-autoscaler-oss-blobstore bucket"
  disabled     = "false"
  project      = var.project
}

resource "google_service_account" "app_autoscaler_releases_uploader" {
  account_id   = "app-autoscaler-releases-upload"
  display_name = "app-autoscaler-releases-uploader"
  description  = "User that can upload to the app-autoscaler-releases bucket"
  disabled     = "false"
  project      = var.project
}


resource "google_storage_bucket" "app_autoscaler_oss_blobstore" {
  name                        = "app-autoscaler-oss-blobstore"
  location                    = var.region
  project                     = var.project
  default_event_based_hold    = "false"
  force_destroy               = "false"
  public_access_prevention    = "inherited"
  requester_pays              = "false"
  storage_class               = "STANDARD"
  uniform_bucket_level_access = "true"
}

resource "google_storage_bucket" "app_autoscaler_releases" {
  name                        = "app-autoscaler-releases"
  location                    = var.region
  project                     = var.project
  default_event_based_hold    = "false"
  force_destroy               = "false"
  public_access_prevention    = "inherited"
  requester_pays              = "false"
  storage_class               = "STANDARD"
  uniform_bucket_level_access = "true"
}
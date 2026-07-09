resource "google_storage_bucket_iam_policy" "app_autoscaler_oss_blobstore" {
  bucket = "b/${google_storage_bucket.app_autoscaler_oss_blobstore.name}"

  policy_data = <<POLICY
{
  "bindings": [
    {
      "members": [
        "projectEditor:${var.project}",
        "projectOwner:${var.project}",
        "serviceAccount:${google_service_account.app_autoscaler_oss_blobstore_uploader.email}",
        "serviceAccount:${google_service_account.app_autoscaler_releases_uploader.email}"
      ],
      "role": "roles/storage.legacyBucketOwner"
    },
    {
      "members": [
        "projectViewer:${var.project}"
      ],
      "role": "roles/storage.legacyBucketReader"
    },
    {
      "members": [
        "projectEditor:${var.project}",
        "projectOwner:${var.project}",
        "serviceAccount:${google_service_account.app_autoscaler_oss_blobstore_uploader.email}"
      ],
      "role": "roles/storage.legacyObjectOwner"
    },
    {
      "members": [
        "projectViewer:${var.project}"
      ],
      "role": "roles/storage.legacyObjectReader"
    },
    {
      "members": [
        "allUsers"
      ],
      "role": "roles/storage.objectViewer"
    }
  ]
}
POLICY
}

resource "google_storage_bucket_iam_policy" "app_autoscaler_releases" {
  bucket = "b/${google_storage_bucket.app_autoscaler_releases.name}"

  policy_data = <<POLICY
{
  "bindings": [
    {
      "members": [
        "projectEditor:${var.project}",
        "projectOwner:${var.project}",
        "serviceAccount:${google_service_account.app_autoscaler_releases_uploader.email}"
      ],
      "role": "roles/storage.legacyBucketOwner"
    },
    {
      "members": [
        "projectViewer:${var.project}"
      ],
      "role": "roles/storage.legacyBucketReader"
    },
    {
      "members": [
        "projectEditor:${var.project}",
        "projectOwner:${var.project}",
        "serviceAccount:${google_service_account.app_autoscaler_releases_uploader.email}"
      ],
      "role": "roles/storage.legacyObjectOwner"
    },
    {
      "members": [
        "allUsers"
      ],
      "role": "roles/storage.legacyObjectReader"
    }
  ]
}
POLICY
}
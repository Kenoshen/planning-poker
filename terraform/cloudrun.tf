resource "google_artifact_registry_repository" "poker" {
  location      = var.region
  repository_id = var.service_name
  format        = "DOCKER"
  description   = "Planning Poker container images"

  depends_on = [google_project_service.services]
}

resource "google_cloud_run_v2_service" "app" {
  name                = var.service_name
  location            = var.region
  deletion_protection = false

  template {
    service_account = google_service_account.cloudrun_runtime.email

    containers {
      # Placeholder — GitHub Actions replaces this on first deploy (image is in lifecycle.ignore_changes)
      image = "us-docker.pkg.dev/cloudrun/container/hello:latest"

      ports {
        container_port = 7878
      }
    }
  }

  # GitHub Actions owns image updates; prevent Terraform from reverting them.
  # Run `terraform apply` to push any other config changes (scaling, etc.)
  lifecycle {
    ignore_changes = [
      template[0].containers[0].image,
      client,
      client_version,
      scaling,
    ]
  }

  depends_on = [google_project_service.services]
}

# Allow unauthenticated public access
resource "google_cloud_run_v2_service_iam_member" "public" {
  name     = google_cloud_run_v2_service.app.name
  location = google_cloud_run_v2_service.app.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}

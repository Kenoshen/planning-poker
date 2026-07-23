# Domain ownership must be verified first (one-time, per Google account):
#   gcloud domains verify yourdomain.com
resource "google_cloud_run_domain_mapping" "poker" {
  count    = var.domain != "" ? 1 : 0
  location = var.region
  name     = var.domain

  metadata {
    namespace = var.project_id
  }

  spec {
    route_name = google_cloud_run_v2_service.app.name
  }

  depends_on = [google_cloud_run_v2_service.app]
}

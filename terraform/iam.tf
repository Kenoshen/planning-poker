# Service account used by GitHub Actions to build, push, and deploy
resource "google_service_account" "github_deploy" {
  account_id   = "github-deploy"
  display_name = "GitHub Actions deploy"
  depends_on   = [google_project_service.services]
}

# Service account the Cloud Run service runs as at runtime
resource "google_service_account" "cloudrun_runtime" {
  account_id   = "cloudrun-runtime"
  display_name = "Cloud Run runtime"
  depends_on   = [google_project_service.services]
}

# github-deploy: push images to Artifact Registry
resource "google_project_iam_member" "deploy_ar_writer" {
  project = var.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${google_service_account.github_deploy.email}"
}

# github-deploy: deploy new Cloud Run revisions
resource "google_project_iam_member" "deploy_run_developer" {
  project = var.project_id
  role    = "roles/run.developer"
  member  = "serviceAccount:${google_service_account.github_deploy.email}"
}

# github-deploy: act as the runtime SA when deploying (required by Cloud Run)
resource "google_service_account_iam_member" "deploy_act_as_runtime" {
  service_account_id = google_service_account.cloudrun_runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.github_deploy.email}"
}

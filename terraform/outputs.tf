output "service_url" {
  description = "Cloud Run service URL"
  value       = google_cloud_run_v2_service.app.uri
}

output "artifact_registry" {
  description = "Full Artifact Registry path — set as GCP_ARTIFACT_REGISTRY in GitHub Actions variables"
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.poker.repository_id}"
}

output "deploy_sa_email" {
  description = "Deploy service account email — use this when creating the GCP_SERVICE_ACCOUNT_KEY"
  value       = google_service_account.github_deploy.email
}

output "domain_mapping_records" {
  description = "DNS records to create at your registrar/DNS host for the custom domain"
  value       = var.domain != "" ? google_cloud_run_domain_mapping.poker[0].status[0].resource_records : []
}

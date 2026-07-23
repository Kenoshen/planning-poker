variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region for all resources"
  type        = string
  default     = "us-central1"
}

variable "service_name" {
  description = "Cloud Run service name"
  type        = string
  default     = "poker"
}

variable "domain" {
  description = "Custom domain to map to the Cloud Run service (leave empty to skip)"
  type        = string
  default     = ""
}

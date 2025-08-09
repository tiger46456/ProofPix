output "project_id" {
  description = "The Google Cloud Project ID"
  value       = var.project_id
}

output "region" {
  description = "The Google Cloud region"
  value       = var.region
}

output "cloud_run_service_url" {
  description = "The URL of the Cloud Run service"
  value       = google_cloud_run_v2_service.proofpix_api.uri
}

output "cloud_run_service_name" {
  description = "The name of the Cloud Run service"
  value       = google_cloud_run_v2_service.proofpix_api.name
}

output "firestore_database_name" {
  description = "The name of the Firestore database"
  value       = google_firestore_database.proofpix_db.name
}

output "storage_bucket_name" {
  description = "The name of the Cloud Storage bucket for asset uploads"
  value       = google_storage_bucket.proofpix_assets_upload.name
}

output "storage_bucket_url" {
  description = "The URL of the Cloud Storage bucket"
  value       = google_storage_bucket.proofpix_assets_upload.url
}

output "service_account_email" {
  description = "The email of the Cloud Run service account"
  value       = google_service_account.proofpix_api.email
} 
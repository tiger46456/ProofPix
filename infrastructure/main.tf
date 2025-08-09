# Terraform configuration
terraform {
  required_version = ">= 1.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.1"
    }
  }
}

# Configure the Google Cloud Provider
provider "google" {
  project = var.project_id
  region  = var.region
  zone    = var.zone
}

# Variables
# variable "organization_id" {
#   description = "The Google Cloud Organization ID (e.g., 123456789012)."
#   type        = string
# }
# Note: Organization ID not needed for projects without an organization

variable "api_image" {
  description = "The container image for the API service."
  type        = string
}

variable "worker_image" {
  description = "The container image for the fingerprint-worker job."
  type        = string
}

# Enable required APIs
resource "google_project_service" "required_apis" {
  for_each = toset([
    "run.googleapis.com",
    "firestore.googleapis.com",
    "storage.googleapis.com",
    "cloudbuild.googleapis.com",
    "cloudtasks.googleapis.com",
    "cloudkms.googleapis.com",
    "spanner.googleapis.com",
    "accesscontextmanager.googleapis.com"
  ])

  project = var.project_id
  service = each.value

  disable_dependent_services = true
  disable_on_destroy         = false
}

# Cloud Run Service
resource "google_cloud_run_v2_service" "proofpix_api" {
  name     = "${var.project_name}-api-${var.environment}"
  location = var.region
  project  = var.project_id

  depends_on = [google_project_service.required_apis]

  template {
    # Configure for managed CPU allocation
    scaling {
      min_instance_count = 0
      max_instance_count = 10
    }

    containers {
      # Container image for the API service
      image = var.api_image
      
      ports {
        container_port = 8080
      }

      # Resource allocation
      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
        cpu_idle = true
        startup_cpu_boost = true
      }

      # Environment variables can be added here
      env {
        name  = "PROJECT_ID"
        value = var.project_id
      }
      
      env {
        name  = "TRILLIAN_LOG_ID"
        value = "1234567890123456789"
      }
      
      env {
        name  = "TRILLIAN_LOG_SERVER_ADDR"
        value = google_cloud_run_v2_service.proofpix_trillian_log_server.uri
      }
    }

    # Service account for the Cloud Run service
    service_account = google_service_account.proofpix_api.email
  }

  traffic {
    percent = 100
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }
}

# Cloud Run Job for Fingerprint Worker
resource "google_cloud_run_v2_job" "proofpix_fingerprint_worker" {
  name     = "proofpix-fingerprint-worker"
  location = var.region
  project  = var.project_id

  depends_on = [google_project_service.required_apis]

  template {
    template {
      containers {
        # Container image for the fingerprint worker
        image = var.worker_image

        # Resource allocation
        resources {
          limits = {
            cpu    = "1"
            memory = "1Gi"
          }
        }

        # Environment variables
        env {
          name  = "PROJECT_ID"
          value = var.project_id
        }
        
        env {
          name  = "TRILLIAN_LOG_ID"
          value = "1234567890123456789"
        }
        
        env {
          name  = "TRILLIAN_LOG_SERVER_ADDR"
          value = google_cloud_run_v2_service.proofpix_trillian_log_server.uri
        }
      }

      # Service account for the fingerprint worker
      service_account = google_service_account.proofpix_signer_sa.email
    }
  }
}

# Cloud Run Service for Trillian Log Server
resource "google_cloud_run_v2_service" "proofpix_trillian_log_server" {
  name     = "proofpix-trillian-log-server"
  location = var.region
  project  = var.project_id
  
  ingress = "INGRESS_TRAFFIC_INTERNAL_ONLY"
  
  depends_on = [google_project_service.required_apis]

  template {
    service_account = google_service_account.proofpix_signer_sa.email
    
    containers {
      image = "gcr.io/trillian-opensource/log_server:latest"
      
      ports {
        container_port = 8091
      }
      
      env {
        name  = "STORAGE_SYSTEM"
        value = "spanner"
      }
      
      env {
        name  = "SPANNER_PROJECT"
        value = var.project_id
      }
      
      env {
        name  = "SPANNER_INSTANCE"
        value = google_spanner_instance.proofpix_trillian_instance.name
      }
      
      env {
        name  = "SPANNER_DATABASE"
        value = google_spanner_database.proofpix_trillian_db.name
      }
      
      env {
        name  = "RPC_ENDPOINT"
        value = "0.0.0.0:8091"
      }
    }
  }

  traffic {
    percent = 100
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }
}

# Cloud Run Service for Trillian Log Signer
resource "google_cloud_run_v2_service" "proofpix_trillian_log_signer" {
  name     = "proofpix-trillian-log-signer"
  location = var.region
  project  = var.project_id
  
  ingress = "INGRESS_TRAFFIC_INTERNAL_ONLY"
  
  depends_on = [google_project_service.required_apis]

  template {
    service_account = google_service_account.proofpix_signer_sa.email
    
    containers {
      image = "gcr.io/trillian-opensource/log_signer:latest"
      
      command = ["--sequencer_interval=1s", "--batch_size=100"]
      
      env {
        name  = "STORAGE_SYSTEM"
        value = "spanner"
      }
      
      env {
        name  = "SPANNER_PROJECT"
        value = var.project_id
      }
      
      env {
        name  = "SPANNER_INSTANCE"
        value = google_spanner_instance.proofpix_trillian_instance.name
      }
      
      env {
        name  = "SPANNER_DATABASE"
        value = google_spanner_database.proofpix_trillian_db.name
      }
      
      env {
        name  = "SIGNER_KMS_KEY_URI"
        value = "gcp-kms://${google_kms_crypto_key.proofpix_trillian_signer.id}"
      }
    }
  }

  traffic {
    percent = 100
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }
}

# Service Account for Cloud Run
resource "google_service_account" "proofpix_api" {
  account_id   = "${var.project_name}-api-${var.environment}"
  display_name = "ProofPix API Service Account"
  description  = "Service account for ProofPix API Cloud Run service"
  project      = var.project_id
}

# Service Account for Fingerprint Worker Signer
resource "google_service_account" "proofpix_signer_sa" {
  account_id   = "proofpix-signer-sa"
  display_name = "ProofPix Fingerprint Worker Signer"
  description  = "Service account for ProofPix fingerprint worker signing operations"
  project      = var.project_id
}

# IAM policy to allow all users to invoke the Cloud Run service
resource "google_cloud_run_service_iam_binding" "public_access" {
  location = google_cloud_run_v2_service.proofpix_api.location
  project  = google_cloud_run_v2_service.proofpix_api.project
  service  = google_cloud_run_v2_service.proofpix_api.name
  role     = "roles/run.invoker"

  members = [
    "allUsers",
  ]
}

# Firestore Database in Native mode
resource "google_firestore_database" "proofpix_db" {
  project     = var.project_id
  name        = "(default)"
  location_id = var.region
  type        = "FIRESTORE_NATIVE"

  depends_on = [google_project_service.required_apis]
}

# Cloud Storage Bucket for asset uploads
resource "google_storage_bucket" "proofpix_assets_upload" {
  name     = "${var.project_name}-assets-upload-${var.environment}-${random_id.bucket_suffix.hex}"
  location = var.region
  project  = var.project_id

  depends_on = [google_project_service.required_apis]

  # Uniform bucket-level access
  uniform_bucket_level_access = true

  # Versioning configuration
  versioning {
    enabled = true
  }

  # Lifecycle management
  lifecycle_rule {
    condition {
      age = 30
    }
    action {
      type = "Delete"
    }
  }

  # CORS configuration for web uploads
  cors {
    origin          = ["*"]
    method          = ["GET", "HEAD", "PUT", "POST", "DELETE"]
    response_header = ["*"]
    max_age_seconds = 3600
  }
}

# Random ID for bucket name uniqueness
resource "random_id" "bucket_suffix" {
  byte_length = 4
}

# Cloud Tasks Queue for fingerprint jobs
resource "google_cloud_tasks_queue" "fingerprint_jobs" {
  name     = "fingerprint-jobs"
  location = var.region
  project  = var.project_id

  depends_on = [google_project_service.required_apis]

  rate_limits {
    max_concurrent_dispatches   = 2
    max_dispatches_per_second   = 5
  }
}

# KMS Key Ring for ProofPix
resource "google_kms_key_ring" "proofpix_keyring" {
  name     = "proofpix-keyring-${var.environment}"
  location = "global"
  project  = var.project_id

  depends_on = [google_project_service.required_apis]
}

# KMS Crypto Key for Trillian Signer
resource "google_kms_crypto_key" "proofpix_trillian_signer" {
  name         = "proofpix-trillian-signer-${var.environment}"
  key_ring     = google_kms_key_ring.proofpix_keyring.id
  purpose      = "ASYMMETRIC_SIGN"

  version_template {
    algorithm = "EC_SIGN_P256_SHA256"
  }

  lifecycle {
    prevent_destroy = true
  }
}

# IAM binding for Cloud Run service to access Firestore
resource "google_project_iam_member" "proofpix_api_firestore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.proofpix_api.email}"
}

# IAM binding for Cloud Run service to access Cloud Storage
resource "google_project_iam_member" "proofpix_api_storage" {
  project = var.project_id
  role    = "roles/storage.objectAdmin"
  member  = "serviceAccount:${google_service_account.proofpix_api.email}"
}

# IAM binding for Cloud Run service to sign URLs for Cloud Storage
resource "google_project_iam_member" "proofpix_api_token_creator" {
  project = var.project_id
  role    = "roles/iam.serviceAccountTokenCreator"
  member  = "serviceAccount:${google_service_account.proofpix_api.email}"
}

# IAM binding for Signer Service Account to use KMS crypto key
resource "google_kms_crypto_key_iam_member" "proofpix_signer_kms" {
  crypto_key_id = google_kms_crypto_key.proofpix_trillian_signer.id
  role          = "roles/cloudkms.cryptoKeySignerVerifier"
  member        = "serviceAccount:${google_service_account.proofpix_signer_sa.email}"
}

# IAM binding for Signer Service Account to access AI Platform (Gemini API)
resource "google_project_iam_member" "proofpix_signer_aiplatform" {
  project = var.project_id
  role    = "roles/aiplatform.user"
  member  = "serviceAccount:${google_service_account.proofpix_signer_sa.email}"
}

# IAM binding for Signer Service Account to access Cloud Storage
resource "google_project_iam_member" "proofpix_signer_storage" {
  project = var.project_id
  role    = "roles/storage.objectAdmin"
  member  = "serviceAccount:${google_service_account.proofpix_signer_sa.email}"
}

# IAM binding for Signer Service Account to access Firestore
resource "google_project_iam_member" "proofpix_signer_firestore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.proofpix_signer_sa.email}"
}

# Google Spanner Instance for Trillian
resource "google_spanner_instance" "proofpix_trillian_instance" {
  config       = "regional-us-central1"
  display_name = "ProofPix Trillian Instance"
  num_nodes    = 1
  project      = var.project_id

  depends_on = [google_project_service.required_apis]
}

# Google Spanner Database for Trillian
resource "google_spanner_database" "proofpix_trillian_db" {
  instance = google_spanner_instance.proofpix_trillian_instance.name
  name     = "proofpix_trillian_db"
  project  = var.project_id

  ddl = [
    "CREATE TABLE Trees (TreeId INT64 NOT NULL, TreeState BYTES(MAX) NOT NULL, TreeType BYTES(MAX) NOT NULL, LogId INT64, CreateTime TIMESTAMP NOT NULL, UpdateTime TIMESTAMP NOT NULL, DisplayName BYTES(MAX), Description BYTES(MAX)) PRIMARY KEY(TreeId)",
    "CREATE TABLE Subtree (TreeId INT64 NOT NULL, SubtreeId BYTES(MAX) NOT NULL, Nodes BYTES(MAX) NOT NULL, SubtreeRevision INT64 NOT NULL) PRIMARY KEY(TreeId, SubtreeId, SubtreeRevision)",
    "CREATE TABLE SequencedLeafData (TreeId INT64 NOT NULL, LeafIdentityHash BYTES(MAX) NOT NULL, MerkleLeafHash BYTES(MAX) NOT NULL, SequenceNumber INT64 NOT NULL, LeafIndex BYTES(MAX), LeafValue BYTES(MAX), ExtraData BYTES(MAX), QueueTimestampNanos INT64) PRIMARY KEY(TreeId, LeafIdentityHash)",
    "CREATE INDEX SequencedLeafDataBySequenceNumber ON SequencedLeafData(TreeId, SequenceNumber)",
    "CREATE TABLE Unsequenced (TreeId INT64 NOT NULL, Bucket INT64 NOT NULL, LeafIdentityHash BYTES(MAX) NOT NULL, MerkleLeafHash BYTES(MAX) NOT NULL, QueueTimestampNanos INT64 NOT NULL) PRIMARY KEY(TreeId, Bucket, QueueTimestampNanos, LeafIdentityHash)",
  ]

  depends_on = [google_spanner_instance.proofpix_trillian_instance]
}

# IAM binding for Signer Service Account to access Spanner Database
resource "google_spanner_database_iam_member" "proofpix_signer_spanner_db" {
  instance = google_spanner_instance.proofpix_trillian_instance.name
  database = google_spanner_database.proofpix_trillian_db.name
  role     = "roles/spanner.databaseUser"
  member   = "serviceAccount:${google_service_account.proofpix_signer_sa.email}"
}

# Access Context Manager requires an Organization
# Commenting out since this project doesn't have an organization

# resource "google_access_context_manager_access_policy" "access_policy" {
#   parent = "organizations/${var.organization_id}"
#   title  = "Default Access Policy"
# }

# resource "google_access_context_manager_service_perimeter" "proofpix_perimeter" {
#   name  = "accessPolicies/${google_access_context_manager_access_policy.access_policy.name}/servicePerimeters/proofpix_perimeter"
#   title = "ProofPix Service Perimeter"
#   
#   spec {
#     restricted_services = [
#       "storage.googleapis.com",
#       "spanner.googleapis.com",
#       "run.googleapis.com",
#       "cloudkms.googleapis.com",
#       "firestore.googleapis.com",
#       "cloudtasks.googleapis.com",
#       "aiplatform.googleapis.com",
#       "artifactregistry.googleapis.com"
#     ]
#   }
#   
#   status {
#     resources = ["projects/${var.project_id}"]
#   }
# } 
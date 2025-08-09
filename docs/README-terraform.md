# ProofPix Infrastructure as Code

This directory contains Terraform configurations to provision the necessary Google Cloud Platform resources for the ProofPix MVP.

## Resources Provisioned

- **Cloud Run Service**: `proofpix-api-{environment}` - Containerized API service
- **Firestore Database**: Native mode database for application data
- **Cloud Storage Bucket**: `proofpix-assets-upload-{environment}` - File upload storage with versioning
- **Service Account**: Dedicated service account with appropriate IAM permissions
- **API Enablement**: Automatically enables required Google Cloud APIs

## Prerequisites

1. **Google Cloud Project**: You need an active GCP project
2. **Terraform**: Install Terraform >= 1.0
3. **Google Cloud CLI**: Install and authenticate with `gcloud auth application-default login`
4. **Billing**: Ensure billing is enabled on your GCP project

## Setup Instructions

1. **Configure Variables**:
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   # Edit terraform.tfvars with your actual project ID
   ```

2. **Initialize Terraform**:
   ```bash
   terraform init
   ```

3. **Plan Deployment**:
   ```bash
   terraform plan
   ```

4. **Apply Configuration**:
   ```bash
   terraform apply
   ```

5. **View Outputs**:
   ```bash
   terraform output
   ```

## Configuration Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `project_id` | Google Cloud Project ID | - | Yes |
| `region` | GCP region for resources | `us-central1` | No |
| `zone` | GCP zone for zonal resources | `us-central1-a` | No |
| `project_name` | Project name prefix | `proofpix` | No |
| `environment` | Deployment environment | `dev` | No |

## Resource Naming Convention

All resources follow the pattern: `{project_name}-{resource_type}-{environment}`

Examples:
- Cloud Run: `proofpix-api-dev`
- Storage Bucket: `proofpix-assets-upload-dev-{random_suffix}`
- Service Account: `proofpix-api-dev@{project_id}.iam.gserviceaccount.com`

## Security Features

- **Service Account**: Dedicated service account with minimal required permissions
- **IAM Bindings**: Firestore and Cloud Storage access for the API service
- **Uniform Bucket Access**: Simplified permission management for Cloud Storage
- **CORS Configuration**: Configured for web-based file uploads

## Cost Optimization

- **Cloud Run**: Scales to zero when not in use
- **Firestore**: Pay-per-operation pricing model
- **Cloud Storage**: Lifecycle policy deletes objects after 30 days
- **Versioning**: Enabled for data protection

## Cleanup

To destroy all resources:
```bash
terraform destroy
```

⚠️ **Warning**: This will permanently delete all data in Firestore and Cloud Storage!

## Next Steps

After successful deployment:
1. Note the Cloud Run service URL from terraform outputs
2. Update your application configuration with the service account email
3. Configure your CI/CD pipeline to deploy to the Cloud Run service
4. Set up monitoring and alerting for production environments 
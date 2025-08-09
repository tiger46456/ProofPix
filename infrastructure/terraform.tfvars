# Example Terraform variables file
# Copy this file to terraform.tfvars and update with your actual values

# Required: Your Google Cloud Project ID
project_id = "make-connection-464709"

# Optional: Customize these values as needed
region       = "us-central1"
zone         = "us-central1-a"
project_name = "proofpix"
environment  = "dev"

# Example values for different environments:
# For development:
# environment = "dev"
# region = "us-central1"

# For staging:
# environment = "staging" 
# region = "us-east1"

# For production:
# environment = "prod"
# region = "us-west1" 
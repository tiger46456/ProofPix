# ProofPix Setup Guide for Beginners

Welcome! This guide will help you install everything needed to run ProofPix, even if you're not a developer.

## 📋 What You'll Install

1. **Go** - Programming language to run your API
2. **Terraform** - Tool to create cloud infrastructure 
3. **Google Cloud CLI** - Tool to manage Google Cloud
4. **Git** - Version control (optional but recommended)
5. **VS Code** - Code editor (optional but helpful)

---

## 🔧 Step-by-Step Installation

### 1. Install Go Programming Language

**What it does**: Runs your ProofPix API server

**Download**: Go to https://golang.org/dl/
1. Click "Download Go for Windows" (should auto-detect)
2. Download the `.msi` file (usually named like `go1.21.5.windows-amd64.msi`)
3. Run the installer and follow the prompts
4. Keep all default settings

**Verify Installation**:
```cmd
# Open Command Prompt (Windows + R, type "cmd", press Enter)
go version
```
You should see something like: `go version go1.21.5 windows/amd64`

---

### 2. Install Terraform

**What it does**: Creates your cloud infrastructure automatically

**Download**: Go to https://www.terraform.io/downloads
1. Click "Windows" under "Binary downloads"
2. Download the `terraform_1.x.x_windows_amd64.zip` file
3. Extract the zip file to a folder like `C:\terraform\`
4. Add to PATH:
   - Press `Windows + R`, type `sysdm.cpl`, press Enter
   - Click "Environment Variables"
   - Under "System Variables", find "Path", click "Edit"
   - Click "New" and add `C:\terraform\` (or wherever you extracted it)
   - Click "OK" on all windows

**Verify Installation**:
```cmd
terraform version
```
You should see: `Terraform v1.x.x`

---

### 3. Install Google Cloud CLI

**What it does**: Lets you manage your Google Cloud account and authenticate

**Download**: Go to https://cloud.google.com/sdk/docs/install
1. Click "Windows" installer
2. Download `GoogleCloudSDKInstaller.exe`
3. Run the installer
4. Follow the prompts (keep default settings)
5. When prompted, choose "Yes" to run `gcloud init`

**Setup**:
```cmd
# This will open a browser for authentication
gcloud auth login

# Set up application default credentials for Terraform
gcloud auth application-default login
```

---

### 4. Install Git (Optional but Recommended)

**What it does**: Tracks changes to your code

**Download**: Go to https://git-scm.com/download/win
1. Download the installer
2. Run it and keep all default settings
3. Choose "Use Git from the Windows Command Prompt" when asked

**Verify Installation**:
```cmd
git --version
```

---

### 5. Install VS Code (Optional but Helpful)

**What it does**: Makes it easier to edit code files

**Download**: Go to https://code.visualstudio.com/
1. Click "Download for Windows"
2. Run the installer
3. Keep all default settings

**Recommended Extensions** (install after opening VS Code):
- Go (by Google)
- Terraform (by HashiCorp)

---

## 🎯 Quick Test

After installing everything, test that it works:

```cmd
# Navigate to your ProofPix folder
cd C:\Users\imadn\Desktop\proofpix

# Test Go
go version

# Test Terraform
terraform version

# Test Google Cloud CLI
gcloud --version

# Build your Go app (this should work now!)
go build -o bin/api cmd/api/main.go

# Run your API
.\bin\api
```

If the API runs, you should see: `ProofPix API server starting on port 8080...`
Visit http://localhost:8080 in your browser to see "Hello World from ProofPix API!"

---

## 🌟 Next Steps After Installation

1. **Create Google Cloud Project**:
   ```cmd
   # Create a new project (replace 'my-proofpix-project' with your desired name)
   gcloud projects create my-proofpix-project --name="ProofPix"
   
   # Set it as your active project
   gcloud config set project my-proofpix-project
   
   # Enable billing (you'll need to do this in the web console)
   ```

2. **Configure Terraform**:
   ```cmd
   # Copy the example configuration
   copy terraform.tfvars.example terraform.tfvars
   
   # Edit terraform.tfvars with your project ID
   notepad terraform.tfvars
   ```

3. **Deploy Infrastructure**:
   ```cmd
   terraform init
   terraform plan
   terraform apply
   ```

---

## 🆘 Troubleshooting

**"Command not found" errors**:
- Restart Command Prompt after installation
- Check that the tool was added to PATH

**Permission errors**:
- Run Command Prompt as Administrator
- Right-click Command Prompt → "Run as administrator"

**Google Cloud authentication issues**:
- Make sure you're logged into the correct Google account
- Re-run: `gcloud auth application-default login`

**Need help?**
- Each tool has excellent documentation
- Google Cloud has a free tier for learning
- Stack Overflow is great for specific error messages

---

## 💡 Tips for Beginners

1. **Take it slow** - Install one tool at a time and test it
2. **Read error messages** - They usually tell you exactly what's wrong
3. **Use the web consoles** - Google Cloud Console is very user-friendly
4. **Start small** - Get the basic setup working before adding complexity
5. **Keep notes** - Write down commands that work for you

You've got this! 🚀 
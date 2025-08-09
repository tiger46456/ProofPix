# 🚀 ProofPix - Complete Guide for Non-Developers

**ProofPix** is your "Authenticity-as-a-Service" application that helps people verify if digital images are real or AI-generated. This guide explains everything you need to know, even if you're not a developer.

---

## 📋 **What Is ProofPix?**

ProofPix is a **professional image authenticity detection system** that:

- 🔍 **Analyzes digital images** to determine if they're real photographs or AI-generated
- 🏅 **Creates verifiable "Authenticity Badges"** for genuine images
- 📝 **Records permanent, tamper-evident entries** in a secure log
- 🌐 **Provides a REST API** that mobile apps, websites, and other tools can use
- 🔐 **Handles user authentication** securely (like Google, Facebook login systems)
- ☁️ **Runs on Google Cloud** for professional-grade scalability and reliability

---

## 🎯 **Quick Start (30 Seconds to Running Server)**

### **✨ The Super Easy Way**
1. **Double-click** `scripts/start-server.bat` 
2. **Wait 5 seconds** for the "🚀 ProofPix API Server Startup" message
3. **Done!** Your server is running at `http://localhost:8080`

### **🔧 Manual Way (If you prefer command line)**
1. **Open PowerShell** in your project folder
2. **Run these commands:**
   ```powershell
   cd scripts
   .\start-server.bat
   ```

### **✅ How to Know It Worked**
You should see:
```
🚀 ProofPix API Server Startup
⚡ Starting your Firebase authentication API...
🌐 Server will start on: http://localhost:8080
📋 Available endpoints: ...
```

---

## 🧪 **Testing Your System (2 Minutes)**

### **🌐 Browser Testing (Recommended)**
1. **Open your web browser**
2. **Go to** `http://localhost:8080` - you should see a welcome message
3. **Go to** `http://localhost:8080/health` - you should see `{"success": true, "message": "OK"}`

### **🔐 Authentication Testing**
1. **Find** `firebase-token-tester.html` in your project folder
2. **Double-click** to open it in your browser
3. **Sign in** with `imad@rafya.store` (your test user)
4. **Click all the test buttons** to verify everything works

### **💻 Command Line Testing**
```powershell
# Test if server is running
Invoke-RestMethod -Uri "http://localhost:8080/health"

# Test public endpoint
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/public"
```

---

## 🏗️ **Project Structure Explained**

Think of your project like a **digital building** with different floors:

### **🎯 Main Application (`cmd/` folder)**
- **`api/`** - The main server that handles all requests (like a reception desk)
- **`fingerprint-worker/`** - The image analysis engine (like a specialized lab)
- **`test-suite/`** - Tools for testing AI detection accuracy
- **`provision-tree/`** - Infrastructure setup tools

### **📚 Documentation (`docs/` folder)**
- **`START-HERE.md`** - Detailed beginner's guide with step-by-step instructions
- **`README-auth.md`** - Everything about user authentication and security
- **`SETUP-GUIDE.md`** - Complete setup instructions
- **`README-terraform.md`** - Cloud infrastructure documentation

### **🔧 Scripts (`scripts/` folder)**
- **`start-server.bat`** - One-click server startup
- **`test-auth.ps1`** - Authentication testing script

### **☁️ Infrastructure (`infrastructure/` folder)**
- **Terraform files** - Automated cloud setup configurations
- **`gcp-credentials.json`** - Google Cloud credentials

### **⚙️ Configuration Files**
- **`go.mod`** - Project dependencies
- **`Makefile`** - Build commands
- **`cloudbuild.yaml`** - Automated deployment instructions
- **`.github/workflows/`** - GitHub automation

---

## 🌐 **Your API Endpoints Explained**

Your ProofPix server provides these services:

| Endpoint | What It Does | Who Can Use It | Example Response |
|----------|--------------|----------------|------------------|
| `GET /` | Welcome message | Everyone | `"Hello World from ProofPix API!"` |
| `GET /health` | Check server status | Everyone | `{"success": true, "message": "OK"}` |
| `GET /api/v1/public` | Public information | Everyone | General app information |
| `GET /api/v1/protected` | Secure user data | Logged-in users only | User-specific data |
| `GET /api/v1/profile` | User profile | Logged-in users only | User details |
| `POST /api/v1/assets` | Upload images for analysis | Logged-in users only | Upload URL + Asset ID |
| `GET /api/v1/admin` | Admin features | Logged-in users only | Admin data |

---

## 🔐 **Authentication System**

ProofPix uses **Firebase Authentication** (Google's professional login system):

### **🎯 How It Works**
1. **Users sign up/log in** through Firebase (like Google login)
2. **Firebase gives them a token** (like a temporary pass)
3. **Your API checks the token** on each request
4. **If valid**, user gets access to protected features

### **🧪 Test Users**
- **Email**: `imad@rafya.store`
- **Use this for testing** all authentication features

---

## ⚡ **Common Tasks & Commands**

### **🛑 Stop the Server**
- **Press `Ctrl+C`** in the PowerShell window where server is running
- **Or close** the PowerShell window

### **🔄 Restart the Server**
1. Stop the server (see above)
2. Double-click `start-server.bat` again

### **🔍 Check Server Status**
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/health"
```

### **🏗️ Rebuild the Application**
```powershell
# In PowerShell, navigate to project folder
go build -o bin/api ./cmd/api
```

### **🧹 Clean Build Files**
```powershell
Remove-Item -Recurse -Force bin/
```

---

## 🚀 **Automated Deployment System**

Your ProofPix system includes **professional-grade automation**:

### **🔄 How It Works**
1. **You make changes** to your code
2. **Push to GitHub** `main` branch
3. **GitHub Actions automatically starts** the deployment process
4. **Google Cloud Build** creates new container images
5. **Terraform applies** any infrastructure changes
6. **New version goes live** on Google Cloud Run within minutes

### **🎯 What This Means**
- **No manual deployment** needed
- **Professional-grade scaling** (handles traffic spikes automatically)
- **Secure cloud hosting** with Google's infrastructure
- **Automatic backups** and monitoring

---

## 🛠️ **Making Changes**

### **🎨 Simple Content Changes**
1. **Open** `cmd/api/main.go` in any text editor
2. **Find** the line with your text (e.g., welcome message)
3. **Change** it to what you want
4. **Save** the file
5. **Restart** the server

### **➕ Adding New Features**
1. **Study** the existing endpoint handlers in `cmd/api/main.go`
2. **Copy** an existing handler function
3. **Modify** it for your needs
4. **Add** the new route to the router
5. **Test** locally before deploying

---

## 🔬 **Image Analysis Features**

ProofPix's core feature is **AI-powered image authenticity detection**:

### **🤖 How It Works**
1. **User uploads** an image through your API
2. **Gemini AI analyzes** the image for signs of AI generation
3. **System generates** an authenticity score (0.0 = definitely AI, 1.0 = definitely real)
4. **Creates** a permanent record in the secure log
5. **Returns** verification results and authenticity badge

### **🧪 Test Image Analysis**
```powershell
# Run the test suite
go run cmd/test-suite/main.go
```

---

## 📱 **Using ProofPix in Other Applications**

### **🌐 From Websites (JavaScript)**
```javascript
// Example API call
const response = await fetch('http://localhost:8080/api/v1/protected', {
  headers: {
    'Authorization': `Bearer ${firebaseToken}`,
    'Content-Type': 'application/json'
  }
});
const data = await response.json();
```

### **📱 From Mobile Apps**
```javascript
// Mobile app integration
fetch('http://localhost:8080/api/v1/profile', {
  headers: {
    'Authorization': 'Bearer ' + userToken,
    'Content-Type': 'application/json'
  }
})
```

### **🔧 From Tools (Postman, etc.)**
- **URL**: `http://localhost:8080/api/v1/protected`
- **Method**: `GET`
- **Headers**: 
  - `Authorization: Bearer YOUR_JWT_TOKEN`
  - `Content-Type: application/json`

---

## 🚨 **Troubleshooting**

### **❌ "Server won't start"**
**Problem**: Port 8080 might be in use  
**Solution**: 
```powershell
# Check what's using port 8080
netstat -an | findstr :8080
# Use different port
$env:PORT='8081'
.\scripts\start-server.bat
```

### **❌ "Failed to fetch" in browser**
**Problem**: Server not running or CORS issue  
**Solution**:
1. Check server is running: `http://localhost:8080/health`
2. Restart the server
3. Clear browser cache

### **❌ "Authentication failed"**
**Problem**: Invalid or expired JWT token  
**Solution**:
1. Get new token from Firebase (tokens expire after 1 hour)
2. Verify you're using correct user credentials
3. Check Firebase project ID in environment variables

### **❌ "Build errors"**
**Problem**: Go compilation issues  
**Solution**:
```powershell
# Check Go installation
go version
# Clean and rebuild
Remove-Item -Recurse -Force bin/
go build -o bin/api ./cmd/api
```

---

## 📖 **Additional Resources**

### **📚 Detailed Documentation**
- **`docs/START-HERE.md`** - Step-by-step beginner's guide
- **`docs/README-auth.md`** - Complete authentication guide
- **`docs/SETUP-GUIDE.md`** - Full setup instructions

### **🌐 External Resources**
- **Firebase Authentication**: [firebase.google.com/docs/auth](https://firebase.google.com/docs/auth)
- **Google Cloud Platform**: [cloud.google.com](https://cloud.google.com)
- **Go Programming**: [golang.org](https://golang.org)

---

## 🎯 **Key Environment Variables**

Your system uses these important settings:

- **`PROJECT_ID`**: `make-connection-464709` (your Google Cloud project)
- **`FIREBASE_PROJECT_ID`**: `make-connection-464709` (your Firebase project)
- **`GCS_BUCKET_NAME`**: `proofpix-assets-upload-dev-e2fecb7f` (your image storage)
- **`PORT`**: `8080` (default server port)

---

## 🎉 **What You've Built**

Congratulations! You now have a **professional-grade image authenticity API** that includes:

✅ **Secure user authentication** (Firebase-based)  
✅ **AI-powered image analysis** (Gemini AI integration)  
✅ **Scalable cloud infrastructure** (Google Cloud Platform)  
✅ **Automated deployment pipeline** (GitHub Actions + Cloud Build)  
✅ **RESTful API** ready for mobile apps and websites  
✅ **Comprehensive documentation** and testing tools  
✅ **Professional monitoring** and logging  

Your ProofPix system is ready to help people verify image authenticity in the age of AI-generated content!
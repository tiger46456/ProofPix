@echo off
set "GOOGLE_APPLICATION_CREDENTIALS=C:\Users\imadn\Desktop\proofpix\infrastructure\gcp-credentials.json"
echo.
echo ====================================
echo    🚀 ProofPix API Server Startup
echo ====================================
echo.
echo ⚡ Starting your Firebase authentication API...
echo.

REM Change to project root directory (parent of scripts folder)
cd /d "%~dp0.."

REM Set required environment variables with quotes for reliability
set "PROJECT_ID=make-connection-464709"
set "FIREBASE_PROJECT_ID=make-connection-464709"
set "GCS_BUCKET_NAME=proofpix-assets-upload-dev-e2fecb7f"

REM Check if API binary exists
if not exist "bin\api.exe" (
    echo 🔨 Building API server...
    go build -o bin/api ./cmd/api
    if errorlevel 1 (
        echo ❌ Build failed! Check your Go installation.
        pause
        exit /b 1
    )
    echo ✅ Build successful!
    echo.
)

echo 🌟 Environment configured:
echo    📡 PROJECT_ID: %PROJECT_ID%
echo    🔥 FIREBASE_PROJECT_ID: %FIREBASE_PROJECT_ID%
echo    🪣 GCS_BUCKET_NAME: %GCS_BUCKET_NAME%
echo    🌐 Server will start on: http://localhost:8080
echo.
echo 📋 Available endpoints:
echo    GET  /                     - Welcome message (public)
echo    GET  /health               - Health check (public)
echo    GET  /api/v1/public        - Public endpoint  
echo    GET  /api/v1/protected     - Protected endpoint (requires auth)
echo    GET  /api/v1/profile       - User profile (requires auth)
echo    POST /api/v1/assets        - Generate upload URL (requires auth)
echo    GET  /api/v1/optional      - Optional auth endpoint
echo    GET  /api/v1/admin         - Admin endpoint (requires auth)
echo.
echo 🧪 To test authentication:
echo    1. Open firebase-token-tester.html in your browser
echo    2. Sign in with: imad@rafya.store
echo    3. Test all endpoints with one click!
echo.
echo 🛑 To stop the server: Press Ctrl+C
echo.
echo ====================================
echo.

REM Start the API server
echo ⏰ Starting server in 3 seconds...
timeout /t 3 /nobreak >nul
echo.
echo 🚀 Server starting...
echo.

REM Start the server directly in the current environment
.\bin\api.exe

REM Keep window open after server stops to show any exit messages
echo.
echo 🛑 Server has stopped. Check above for any error messages.
pause
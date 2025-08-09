# ProofPix Firebase Authentication

This document explains how Firebase Authentication is implemented in the ProofPix API using the official Firebase Admin Go SDK.

## Overview

The authentication system provides:
- JWT token verification using Firebase Admin SDK
- Middleware for protected routes
- Optional authentication for flexible endpoints
- User context extraction for authenticated requests
- Proper error handling with JSON responses

## Implementation

### Core Components

**`internal/auth/middleware.go`** - Main authentication logic:
- `VerifyFirebaseJWT()` - Strict authentication middleware
- `OptionalFirebaseJWT()` - Optional authentication middleware  
- `GetUserID()` & `GetUser()` - Context extraction helpers
- `InitFirebase()` - Firebase client initialization

### Middleware Usage

#### 1. Required Authentication
```go
// Requires valid Firebase JWT token
mux.Handle("/api/v1/protected", auth.VerifyFirebaseJWT(http.HandlerFunc(handleProtected)))
```

#### 2. Optional Authentication  
```go
// Works with or without authentication
mux.Handle("/api/v1/optional", auth.OptionalFirebaseJWT(http.HandlerFunc(handleOptional)))
```

#### 3. Extract User Information
```go
func handleProtected(w http.ResponseWriter, r *http.Request) {
    userID, ok := auth.GetUserID(r)
    if !ok {
        // Handle error - user ID not found
        return
    }
    
    user, ok := auth.GetUser(r)
    if !ok {
        // Handle error - user token not found
        return
    }
    
    // Access user claims
    email := user.Claims["email"]
    emailVerified := user.Claims["email_verified"]
}
```

## API Endpoints

### Public Endpoints (No Authentication)
- `GET /` - Root endpoint
- `GET /health` - Health check
- `GET /api/v1/public` - Public API endpoint

### Protected Endpoints (Authentication Required)
- `GET /api/v1/protected` - Protected endpoint demo
- `GET /api/v1/profile` - User profile information
- `GET /api/v1/admin` - Admin endpoint (can add role checks)

### Optional Authentication Endpoints
- `GET /api/v1/optional` - Works with or without auth

## Configuration

### Environment Variables

**Required:**
- `PROJECT_ID` or `FIREBASE_PROJECT_ID` - Your Firebase project ID

**Optional:**
- `FIREBASE_SERVICE_ACCOUNT_KEY` - Service account JSON (for local development)
- `PORT` - Server port (default: 8080)

### Firebase Setup

1. **Enable Authentication** in Firebase Console
2. **Create Service Account** (for local development):
   ```bash
   # Generate service account key
   gcloud iam service-accounts keys create key.json \
     --iam-account=proofpix-api-dev@make-connection-464709.iam.gserviceaccount.com
   ```

3. **Configure Authentication Methods** in Firebase Console:
   - Email/Password
   - Google Sign-In
   - Other providers as needed

## Testing Authentication

### 1. Get Firebase Auth Token

**Frontend JavaScript:**
```javascript
import { getAuth, signInWithEmailAndPassword } from 'firebase/auth';

const auth = getAuth();
const userCredential = await signInWithEmailAndPassword(auth, email, password);
const token = await userCredential.user.getIdToken();
```

**cURL Testing:**
```bash
# Without authentication (should fail)
curl https://your-api-url/api/v1/protected

# With authentication
curl -H "Authorization: Bearer YOUR_FIREBASE_TOKEN" \
     https://your-api-url/api/v1/protected
```

### 2. Test Public Endpoints
```bash
# Public endpoint (no auth needed)
curl https://your-api-url/api/v1/public

# Optional auth endpoint (works both ways)
curl https://your-api-url/api/v1/optional
curl -H "Authorization: Bearer YOUR_TOKEN" https://your-api-url/api/v1/optional
```

## Error Responses

### Authentication Errors
```json
{
  "error": "Missing Authorization header",
  "message": "Authorization header is required"
}
```

```json
{
  "error": "Invalid token", 
  "message": "Token verification failed"
}
```

### Success Response
```json
{
  "success": true,
  "message": "Access granted to protected endpoint",
  "data": {
    "endpoint": "protected",
    "auth_required": "true", 
    "user_id": "firebase-user-id",
    "user_claims": {
      "email": "user@example.com",
      "email_verified": true
    }
  }
}
```

## Security Features

- **Token Verification**: All tokens verified against Firebase Auth
- **Context Isolation**: User data stored in request context
- **Error Handling**: Secure error messages without token exposure
- **Header Validation**: Proper "Bearer TOKEN" format enforcement
- **Application Default Credentials**: Seamless Cloud Run integration

## Local Development

1. **Install Dependencies:**
   ```bash
   go mod tidy
   ```

2. **Set Environment Variables:**
   ```bash
   export PROJECT_ID=make-connection-464709
   export FIREBASE_PROJECT_ID=make-connection-464709
   ```

3. **Run Server:**
   ```bash
   go run cmd/api/main.go
   ```

4. **Test Endpoints:**
   ```bash
   curl http://localhost:8080/health
   curl http://localhost:8080/api/v1/public
   ```

## Production Deployment

The authentication is already configured for your Cloud Run deployment:
- `PROJECT_ID` environment variable is automatically set by Terraform
- Application Default Credentials are used for Firebase initialization
- No additional configuration needed

Your Firebase authentication is ready to use! 🚀

## Advanced Usage

### Custom Claims & Roles

Add custom claims in Firebase Console or via Admin SDK:
```javascript
// In your Firebase Functions or Admin SDK
await admin.auth().setCustomUserClaims(uid, { role: 'admin', permissions: ['read', 'write'] });
```

Access in Go:
```go
user, _ := auth.GetUser(r)
if customClaims, exists := user.Claims["custom_claims"]; exists {
    if claims, ok := customClaims.(map[string]interface{}); ok {
        role := claims["role"]
        permissions := claims["permissions"]
    }
}
```

### Middleware Chaining

```go
// Combine multiple middlewares
mux.Handle("/api/v1/secure", 
    corsMiddleware(
        rateLimitMiddleware(
            auth.VerifyFirebaseJWT(
                http.HandlerFunc(handleSecure)
            )
        )
    )
)
``` 
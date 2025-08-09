package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"github.com/google/trillian"
	"github.com/google/uuid"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"proofpix/internal/auth"
)

// Response represents a JSON response
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// UserResponse represents a user response
type UserResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email,omitempty"`
}

// AssetResponse represents an asset upload response
type AssetResponse struct {
	AssetID   string `json:"asset_id"`
	UploadURL string `json:"upload_url"`
}

// Asset represents an image asset with its analysis results
type Asset struct {
	ID               string    `firestore:"id"`
	UserID           string    `firestore:"user_id"`
	Status           string    `firestore:"status"`
	CreatedAt        time.Time `firestore:"created_at"`
	RawAnalysis      string    `firestore:"raw_analysis"`
	OriginalityScore int       `firestore:"originality_score"`
	Narrative        string    `firestore:"narrative"`
	Embedding        []float32 `firestore:"embedding"`
	TrillianLeafIndex int64    `firestore:"trillian_leaf_index,omitempty"`
}

func main() {
	// Initialize Firebase
	if err := auth.InitFirebase(); err != nil {
		log.Fatalf("Failed to initialize Firebase: %v", err)
	}

	// Setup routes with CORS middleware
	mux := http.NewServeMux()
	
	// Configure CORS middleware with rs/cors library  
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		},
		AllowedHeaders: []string{
			"*", // Allow all headers for development
		},
		ExposedHeaders: []string{
			"Content-Length",
			"Content-Type",
		},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
		Debug:            true,
	})
	
	// Wrap mux with CORS middleware
	handler := c.Handler(mux)

	// Public routes (no authentication required)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Simple test handler called for path: %s", r.URL.Path)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("TEST HANDLER WORKING!"))
	})
	mux.HandleFunc("/api/v1/public", handlePublic)
	mux.HandleFunc("/api/v1/verify/", verifyHandler)
	
	// Handle root path specifically (not as catch-all)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only handle exact root path
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		handleRoot(w, r)
	})

	// Protected routes (authentication required)
	mux.Handle("/api/v1/protected", auth.VerifyFirebaseJWT(http.HandlerFunc(handleProtected)))
	mux.Handle("/api/v1/profile", auth.VerifyFirebaseJWT(http.HandlerFunc(handleProfile)))
    mux.Handle("/api/v1/assets", auth.VerifyFirebaseJWT(http.HandlerFunc(handleAssets)))
    mux.Handle("/api/v1/assets/", auth.VerifyFirebaseJWT(http.HandlerFunc(handleAssets)))

	// Optional authentication routes (works with or without auth)
	mux.Handle("/api/v1/optional", auth.OptionalFirebaseJWT(http.HandlerFunc(handleOptional)))

	// Admin routes (protected + additional checks can be added)
	mux.Handle("/api/v1/admin", auth.VerifyFirebaseJWT(http.HandlerFunc(handleAdmin)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("ProofPix API server starting on port %s...\n", port)
	fmt.Println("Available endpoints:")
	fmt.Println("  GET  /                     - Root endpoint (public)")
	fmt.Println("  GET  /health               - Health check (public)")
	fmt.Println("  GET  /api/v1/public        - Public endpoint")
	fmt.Println("  GET  /api/v1/verify/{id}   - Asset verification (public)")
	fmt.Println("  GET  /api/v1/protected     - Protected endpoint (requires auth)")
	fmt.Println("  GET  /api/v1/profile       - User profile (requires auth)")
	fmt.Println("  POST /api/v1/assets        - Generate upload URL (requires auth)")
	fmt.Println("  GET  /api/v1/optional      - Optional auth endpoint")
	fmt.Println("  GET  /api/v1/admin         - Admin endpoint (requires auth)")
	
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// handleRoot handles the root endpoint
func handleRoot(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleRoot called for path: %s", r.URL.Path)
	
	// Only handle exact root path, not all unmatched paths
	if r.URL.Path != "/" {
		log.Printf("handleRoot rejecting path: %s", r.URL.Path)
		http.NotFound(w, r)
		return
	}
	
	response := Response{
		Success: true,
		Message: "Hello World from ProofPix API!",
		Data: map[string]string{
			"version": "1.0.0",
			"service": "proofpix-api",
		},
	}
	respondJSON(w, http.StatusOK, response)
}

// handleHealth handles the health check endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
	response := Response{
		Success: true,
		Message: "OK",
		Data: map[string]string{
			"status": "healthy",
		},
	}
	respondJSON(w, http.StatusOK, response)
}

// handleTest serves the Firebase token tester HTML page
func handleTest(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleTest called for path: %s", r.URL.Path)
	
	const testHTML = `<!DOCTYPE html>
<html>
<head>
    <title>🔐 ProofPix Firebase Token Tester</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; background: #f8f9fa; }
        .container { background: white; padding: 30px; border-radius: 12px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
        .header { text-align: center; margin-bottom: 30px; }
        .section { margin: 20px 0; padding: 20px; background: #f8f9fa; border-radius: 8px; }
        input, button { padding: 12px; margin: 8px 0; width: 100%; box-sizing: border-box; border: 1px solid #ddd; border-radius: 6px; }
        button { background: #4285f4; color: white; border: none; cursor: pointer; font-weight: bold; }
        button:hover { background: #3367d6; }
        button:disabled { background: #ccc; cursor: not-allowed; }
        .token-display { background: #e8f5e8; padding: 15px; border-radius: 6px; margin: 10px 0; word-break: break-all; font-family: monospace; font-size: 12px; }
        .api-result { background: #f0f8ff; padding: 15px; border-radius: 6px; margin: 10px 0; }
        .success { color: #28a745; }
        .error { color: #dc3545; }
        .info { color: #17a2b8; }
        .warning { color: #ffc107; }
        pre { background: #f8f9fa; padding: 10px; border-radius: 4px; overflow-x: auto; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔐 ProofPix Firebase Authentication Tester</h1>
            <p>Test your Firebase JWT authentication with your Go API</p>
        </div>

        <div class="section">
            <h3>📧 Step 1: Sign In to Firebase</h3>
            <input type="email" id="email" placeholder="Email" value="imad@rafya.store">
            <input type="password" id="password" placeholder="Password">
            <button id="signInBtn" onclick="signInUser()">🔑 Sign In & Get Token</button>
            <div id="signInResult"></div>
        </div>

        <div class="section">
            <h3>🎟️ Step 2: JWT Token</h3>
            <div id="tokenDisplay" style="display: none;">
                <strong>Your JWT Token:</strong>
                <div class="token-display" id="tokenValue"></div>
                <button onclick="copyToken()">📋 Copy Token</button>
            </div>
        </div>

        <div class="section">
            <h3>🧪 Step 3: Test API Endpoints</h3>
            <button onclick="testPublicEndpoint()">🌐 Test Public Endpoint</button>
            <button onclick="testProtectedEndpoint()" id="testProtectedBtn" disabled>🔒 Test Protected Endpoint</button>
            <button onclick="testProfileEndpoint()" id="testProfileBtn" disabled>👤 Test Profile Endpoint</button>
            <button onclick="testOptionalEndpoint()">🔄 Test Optional Auth Endpoint</button>
            <div id="apiResults"></div>
        </div>

        <div class="section">
            <h3>📋 Instructions</h3>
            <ol>
                <li>Enter your Firebase user credentials (imad@rafya.store)</li>
                <li>Click "Sign In & Get Token" to authenticate and get your JWT</li>
                <li>Use the buttons to test different API endpoints</li>
                <li>Copy the token to use with curl or other tools</li>
            </ol>
        </div>
    </div>

    <!-- Firebase SDK -->
    <script type="module">
        import { initializeApp } from 'https://www.gstatic.com/firebasejs/10.12.2/firebase-app.js';
        import { getAuth, signInWithEmailAndPassword, onAuthStateChanged } from 'https://www.gstatic.com/firebasejs/10.12.2/firebase-auth.js';

        // Your Firebase configuration
        const firebaseConfig = {
            apiKey: "AIzaSyBDXrEZjMdDDdGbHpHRs0I0xxO1K3XGNTA",
            authDomain: "make-connection-464709.firebaseapp.com",
            projectId: "make-connection-464709",
            storageBucket: "make-connection-464709.firebasestorage.app",
            messagingSenderId: "992932754864",
            appId: "1:992932754864:web:ba25cd7c803a1fe7929275",
            measurementId: "G-8LG9ZBJWC5"
        };

        // Initialize Firebase
        const app = initializeApp(firebaseConfig);
        const auth = getAuth(app);

        let currentToken = null;

        // Make functions global
        window.signInUser = async function() {
            const email = document.getElementById('email').value;
            const password = document.getElementById('password').value;
            const resultDiv = document.getElementById('signInResult');
            const signInBtn = document.getElementById('signInBtn');
            
            if (!email || !password) {
                resultDiv.innerHTML = '<p class="error">❌ Please enter both email and password</p>';
                return;
            }

            try {
                signInBtn.disabled = true;
                signInBtn.textContent = '🔄 Signing in...';
                resultDiv.innerHTML = '<p class="info">🔄 Authenticating with Firebase...</p>';
                
                const userCredential = await signInWithEmailAndPassword(auth, email, password);
                const user = userCredential.user;
                const token = await user.getIdToken();
                
                currentToken = token;
                
                resultDiv.innerHTML = ` + "`" + `
                    <div class="success">
                        <p>✅ Successfully signed in!</p>
                        <p><strong>Email:</strong> ${user.email}</p>
                        <p><strong>UID:</strong> ${user.uid}</p>
                        <p><strong>Email Verified:</strong> ${user.emailVerified}</p>
                    </div>
                ` + "`" + `;
                
                // Show token
                document.getElementById('tokenDisplay').style.display = 'block';
                document.getElementById('tokenValue').textContent = token;
                
                // Enable protected endpoint buttons
                document.getElementById('testProtectedBtn').disabled = false;
                document.getElementById('testProfileBtn').disabled = false;
                
            } catch (error) {
                resultDiv.innerHTML = ` + "`" + `<p class="error">❌ Sign-in failed: ${error.message}</p>` + "`" + `;
            } finally {
                signInBtn.disabled = false;
                signInBtn.textContent = '🔑 Sign In & Get Token';
            }
        };

        window.copyToken = function() {
            const tokenValue = document.getElementById('tokenValue').textContent;
            navigator.clipboard.writeText(tokenValue).then(() => {
                alert('✅ Token copied to clipboard!');
            });
        };

        window.testPublicEndpoint = async function() {
            await testEndpoint('http://localhost:8080/api/v1/public', false);
        };

        window.testProtectedEndpoint = async function() {
            await testEndpoint('http://localhost:8080/api/v1/protected', true);
        };

        window.testProfileEndpoint = async function() {
            await testEndpoint('http://localhost:8080/api/v1/profile', true);
        };

        window.testOptionalEndpoint = async function() {
            await testEndpoint('http://localhost:8080/api/v1/optional', currentToken ? true : false);
        };

        async function testEndpoint(url, useAuth) {
            const resultsDiv = document.getElementById('apiResults');
            const headers = {
                'Content-Type': 'application/json'
            };
            
            if (useAuth && currentToken) {
                headers['Authorization'] = ` + "`" + `Bearer ${currentToken}` + "`" + `;
            }

            try {
                resultsDiv.innerHTML += ` + "`" + `<div class="info">🔄 Testing: ${url}</div>` + "`" + `;
                
                const response = await fetch(url, { headers });
                const data = await response.json();
                
                const statusClass = response.ok ? 'success' : 'error';
                const statusIcon = response.ok ? '✅' : '❌';
                
                resultsDiv.innerHTML += ` + "`" + `
                    <div class="api-result">
                        <div class="${statusClass}">
                            <strong>${statusIcon} ${url}</strong> - Status: ${response.status}
                        </div>
                        <pre>${JSON.stringify(data, null, 2)}</pre>
                    </div>
                ` + "`" + `;
                
            } catch (error) {
                resultsDiv.innerHTML += ` + "`" + `
                    <div class="api-result">
                        <div class="error">
                            <strong>❌ ${url}</strong> - Error: ${error.message}
                        </div>
                    </div>
                ` + "`" + `;
            }
        }

        // Auto-detect if user is already signed in
        onAuthStateChanged(auth, async (user) => {
            if (user) {
                console.log('User already signed in:', user.email);
                // Automatically get token if user is already signed in
                currentToken = await user.getIdToken();
                document.getElementById('tokenDisplay').style.display = 'block';
                document.getElementById('tokenValue').textContent = currentToken;
                document.getElementById('testProtectedBtn').disabled = false;
                document.getElementById('testProfileBtn').disabled = false;
            }
        });
    </script>
</body>
</html>`
	
	// Set content type and serve the HTML
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(testHTML))
}

// handlePublic handles public endpoints
func handlePublic(w http.ResponseWriter, r *http.Request) {
	response := Response{
		Success: true,
		Message: "This is a public endpoint accessible to everyone",
		Data: map[string]string{
			"endpoint": "public",
			"auth_required": "false",
		},
	}
	respondJSON(w, http.StatusOK, response)
}

// handleProtected handles protected endpoints that require authentication
func handleProtected(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		respondError(w, http.StatusInternalServerError, "User ID not found in context")
		return
	}

	user, ok := auth.GetUser(r)
	if !ok {
		respondError(w, http.StatusInternalServerError, "User not found in context")
		return
	}

	response := Response{
		Success: true,
		Message: "Access granted to protected endpoint",
		Data: map[string]interface{}{
			"endpoint": "protected",
			"auth_required": "true",
			"user_id": userID,
			"user_claims": map[string]interface{}{
				"email": user.Claims["email"],
				"email_verified": user.Claims["email_verified"],
			},
		},
	}
	respondJSON(w, http.StatusOK, response)
}

// handleProfile handles user profile endpoints
func handleProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		respondError(w, http.StatusInternalServerError, "User ID not found in context")
		return
	}

	user, ok := auth.GetUser(r)
	if !ok {
		respondError(w, http.StatusInternalServerError, "User not found in context")
		return
	}

	userResponse := UserResponse{
		UserID: userID,
	}

	if email, exists := user.Claims["email"]; exists {
		if emailStr, ok := email.(string); ok {
			userResponse.Email = emailStr
		}
	}

	response := Response{
		Success: true,
		Message: "User profile retrieved successfully",
		Data:    userResponse,
	}
	respondJSON(w, http.StatusOK, response)
}

// handleOptional handles endpoints with optional authentication
func handleOptional(w http.ResponseWriter, r *http.Request) {
	userID, authenticated := auth.GetUserID(r)
	
	data := map[string]interface{}{
		"endpoint": "optional",
		"authenticated": authenticated,
	}

	if authenticated {
		data["user_id"] = userID
		data["message"] = "Hello authenticated user!"
	} else {
		data["message"] = "Hello anonymous user!"
	}

	response := Response{
		Success: true,
		Message: "Optional authentication endpoint",
		Data:    data,
	}
	respondJSON(w, http.StatusOK, response)
}

// handleAdmin handles admin endpoints (can add additional role checks here)
func handleAdmin(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		respondError(w, http.StatusInternalServerError, "User ID not found in context")
		return
	}

	user, ok := auth.GetUser(r)
	if !ok {
		respondError(w, http.StatusInternalServerError, "User not found in context")
		return
	}

	// Here you could add additional admin role checks
	// For example, check if user has admin role in custom claims
	isAdmin := false
	if customClaims, exists := user.Claims["custom_claims"]; exists {
		if claims, ok := customClaims.(map[string]interface{}); ok {
			if role, exists := claims["role"]; exists {
				isAdmin = role == "admin"
			}
		}
	}

	response := Response{
		Success: true,
		Message: "Admin endpoint accessed",
		Data: map[string]interface{}{
			"endpoint": "admin",
			"user_id": userID,
			"is_admin": isAdmin,
			"note": "Add custom claims to Firebase user for role-based access",
		},
	}
	respondJSON(w, http.StatusOK, response)
}

// handleAssets handles asset upload requests by generating pre-signed URLs
func handleAssets(w http.ResponseWriter, r *http.Request) {
	// Only allow POST method
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get authenticated user ID from context (added by middleware)
	userID, ok := auth.GetUserID(r)
	if !ok {
		respondError(w, http.StatusInternalServerError, "User ID not found in context")
		return
	}

	// Generate a new unique asset ID
	assetID := uuid.New().String()

	// Construct object name: uploads/{userID}/{assetID}.jpg
	objectName := fmt.Sprintf("uploads/%s/%s.jpg", userID, assetID)

	// Get bucket name from environment variable
	bucketName := os.Getenv("GCS_BUCKET_NAME")
	if bucketName == "" {
		log.Printf("GCS_BUCKET_NAME environment variable not set")
		respondError(w, http.StatusInternalServerError, "Storage configuration error")
		return
	}

	// Create Google Cloud Storage client
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Printf("Failed to create storage client: %v", err)
		respondError(w, http.StatusInternalServerError, "Storage service unavailable")
		return
	}
	defer client.Close()

	// Get bucket handle
	bucket := client.Bucket(bucketName)

	// Generate signed URL for PUT operation
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "PUT",
		Headers: []string{
			"Content-Type:image/jpeg",
		},
		Expires: time.Now().Add(15 * time.Minute), // 15 minutes expiry
	}

	uploadURL, err := bucket.SignedURL(objectName, opts)
	if err != nil {
		log.Printf("Failed to generate signed URL: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to generate upload URL")
		return
	}

	// Create response with asset ID and upload URL
	assetResponse := AssetResponse{
		AssetID:   assetID,
		UploadURL: uploadURL,
	}

	response := Response{
		Success: true,
		Message: "Upload URL generated successfully",
		Data:    assetResponse,
	}

	respondJSON(w, http.StatusOK, response)
}

// verifyHandler handles asset verification requests
func verifyHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow GET method
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse assetID from URL path
	// Expected path: /api/v1/verify/{assetID}
	path := r.URL.Path
	const prefix = "/api/v1/verify/"
	
	if !strings.HasPrefix(path, prefix) {
		respondError(w, http.StatusBadRequest, "Invalid verify path")
		return
	}
	
	assetID := strings.TrimPrefix(path, prefix)
	if assetID == "" {
		respondError(w, http.StatusBadRequest, "Asset ID is required")
		return
	}
	
	// Log the assetID to console
	log.Printf("Verify request received for assetID: %s", assetID)
	
	// Get project ID from environment
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Printf("GOOGLE_CLOUD_PROJECT environment variable not set")
		respondError(w, http.StatusInternalServerError, "Server configuration error")
		return
	}
	
	// Initialize Firestore client
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Printf("Failed to create Firestore client: %v", err)
		respondError(w, http.StatusInternalServerError, "Database service unavailable")
		return
	}
	defer client.Close()
	
	// Fetch the asset document from Firestore
	docRef := client.Collection("assets").Doc(assetID)
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		if firestore.IsNotFound(err) {
			log.Printf("Asset not found: %s", assetID)
			respondError(w, http.StatusNotFound, "Asset not found")
			return
		}
		log.Printf("Failed to fetch asset %s: %v", assetID, err)
		respondError(w, http.StatusInternalServerError, "Failed to fetch asset")
		return
	}
	
	// Unmarshal the document data into Asset struct
	var asset Asset
	if err := docSnap.DataTo(&asset); err != nil {
		log.Printf("Failed to unmarshal asset %s: %v", assetID, err)
		respondError(w, http.StatusInternalServerError, "Failed to parse asset data")
		return
	}
	
	// Check if asset has been logged to Trillian
	if asset.TrillianLeafIndex == 0 {
		response := Response{
			Success: true,
			Message: "Asset found but not yet included in the log",
			Data: map[string]interface{}{
				"asset_id": assetID,
				"status":   "pending_inclusion",
				"logged":   false,
			},
		}
		respondJSON(w, http.StatusAccepted, response)
		return
	}
	
	// Asset has been logged - get inclusion proof from Trillian
	trillianLogID := os.Getenv("TRILLIAN_LOG_ID")
	if trillianLogID == "" {
		log.Printf("TRILLIAN_LOG_ID environment variable not set")
		respondError(w, http.StatusInternalServerError, "Server configuration error")
		return
	}
	
	logID, err := strconv.ParseInt(trillianLogID, 10, 64)
	if err != nil {
		log.Printf("Failed to parse TRILLIAN_LOG_ID: %v", err)
		respondError(w, http.StatusInternalServerError, "Server configuration error")
		return
	}
	
	// Call getInclusionProof function
	inclusionProofResponse, err := getInclusionProof(ctx, logID, asset.TrillianLeafIndex)
	if err != nil {
		log.Printf("Failed to get inclusion proof for asset %s: %v", assetID, err)
		respondError(w, http.StatusInternalServerError, "Failed to retrieve inclusion proof")
		return
	}
	
	// Set Content-Type header to application/json
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	// Marshal the inclusion proof response to JSON and write it
	if err := json.NewEncoder(w).Encode(inclusionProofResponse); err != nil {
		log.Printf("Error encoding inclusion proof response to JSON: %v", err)
		// Response headers already sent, so we can't change status code
		return
	}
}

// getInclusionProof retrieves an inclusion proof from the Trillian log server
func getInclusionProof(ctx context.Context, logID int64, leafIndex int64) (*trillian.GetInclusionProofResponse, error) {
	// Read TRILLIAN_LOG_SERVER_ADDR from environment variable
	logServerAddr := os.Getenv("TRILLIAN_LOG_SERVER_ADDR")
	if logServerAddr == "" {
		return nil, fmt.Errorf("TRILLIAN_LOG_SERVER_ADDR environment variable not set")
	}
	
	// Establish a secure gRPC connection to the server
	log.Printf("Establishing gRPC connection to Trillian Log Server at %s", logServerAddr)
	conn, err := grpc.DialContext(ctx, logServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Trillian Log Server at %s: %v", logServerAddr, err)
	}
	
	// Ensure the gRPC connection is properly closed
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			log.Printf("Error closing gRPC connection: %v", closeErr)
		}
	}()
	
	// Create a trillian.TrillianLogClient
	client := trillian.NewTrillianLogClient(conn)
	
	// Construct and send a trillian.GetInclusionProofRequest
	request := &trillian.GetInclusionProofRequest{
		LogId:     logID,
		LeafIndex: leafIndex,
	}
	
	log.Printf("Requesting inclusion proof for log %d, leaf index %d", logID, leafIndex)
	response, err := client.GetInclusionProof(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get inclusion proof from Trillian log %d for leaf %d: %v", logID, leafIndex, err)
	}
	
	log.Printf("Successfully retrieved inclusion proof for log %d, leaf index %d", logID, leafIndex)
	return response, nil
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}



// respondError sends an error response
func respondError(w http.ResponseWriter, statusCode int, message string) {
	response := Response{
		Success: false,
		Message: message,
	}
	respondJSON(w, statusCode, response)
} 
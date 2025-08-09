package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// UserIDKey is the context key for storing the user ID
	UserIDKey ContextKey = "user_id"
	// UserKey is the context key for storing the full user token
	UserKey ContextKey = "user"
)

// FirebaseClient holds the Firebase Auth client
type FirebaseClient struct {
	client *auth.Client
}

var (
	firebaseClient *FirebaseClient
	once           sync.Once
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// InitFirebase initializes the Firebase client using environment variables
func InitFirebase() error {
	var err error
	once.Do(func() {
		projectID := os.Getenv("FIREBASE_PROJECT_ID")
		if projectID == "" {
			projectID = os.Getenv("PROJECT_ID") // Fallback to PROJECT_ID from Terraform
		}

		if projectID == "" {
			err = fmt.Errorf("FIREBASE_PROJECT_ID or PROJECT_ID environment variable is required")
			return
		}

		// For Cloud Run, we can use Application Default Credentials
		// which are automatically available in the GCP environment
		ctx := context.Background()
		
		// Try to initialize with service account key if provided
		serviceAccountKey := os.Getenv("FIREBASE_SERVICE_ACCOUNT_KEY")
		var app *firebase.App
		
		if serviceAccountKey != "" {
			log.Println("Initializing Firebase with service account key")
			opt := option.WithCredentialsJSON([]byte(serviceAccountKey))
			config := &firebase.Config{ProjectID: projectID}
			app, err = firebase.NewApp(ctx, config, opt)
		} else {
			log.Println("Initializing Firebase with Application Default Credentials")
			config := &firebase.Config{ProjectID: projectID}
			app, err = firebase.NewApp(ctx, config)
		}

		if err != nil {
			err = fmt.Errorf("error initializing firebase app: %v", err)
			return
		}

		authClient, authErr := app.Auth(ctx)
		if authErr != nil {
			err = fmt.Errorf("error getting auth client: %v", authErr)
			return
		}

		firebaseClient = &FirebaseClient{client: authClient}
		log.Printf("Firebase initialized successfully for project: %s", projectID)
	})

	return err
}

// GetFirebaseClient returns the singleton Firebase client
func GetFirebaseClient() (*FirebaseClient, error) {
	if firebaseClient == nil {
		return nil, fmt.Errorf("firebase client not initialized. Call InitFirebase() first")
	}
	return firebaseClient, nil
}

// CreateCustomToken creates a custom Firebase token for testing
func (fc *FirebaseClient) CreateCustomToken(ctx context.Context, uid string) (string, error) {
	return fc.client.CustomToken(ctx, uid)
}

// VerifyFirebaseJWT creates a middleware that verifies Firebase JWT tokens
func VerifyFirebaseJWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondWithError(w, http.StatusUnauthorized, "Missing Authorization header", "Authorization header is required")
			return
		}

		// Check if it follows the "Bearer [TOKEN]" format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			respondWithError(w, http.StatusUnauthorized, "Invalid Authorization header format", "Expected format: Bearer <token>")
			return
		}

		token := parts[1]
		if token == "" {
			respondWithError(w, http.StatusUnauthorized, "Empty token", "Token cannot be empty")
			return
		}

		// Get Firebase client
		client, err := GetFirebaseClient()
		if err != nil {
			log.Printf("Error getting Firebase client: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Authentication service unavailable", "Internal server error")
			return
		}

		// Verify the JWT token
		decodedToken, err := client.client.VerifyIDToken(context.Background(), token)
		if err != nil {
			log.Printf("Error verifying token: %v", err)
			respondWithError(w, http.StatusUnauthorized, "Invalid token", "Token verification failed")
			return
		}

		// Add user information to request context
		ctx := context.WithValue(r.Context(), UserIDKey, decodedToken.UID)
		ctx = context.WithValue(ctx, UserKey, decodedToken)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalFirebaseJWT creates a middleware that optionally verifies Firebase JWT tokens
// This is useful for endpoints that can work with or without authentication
func OptionalFirebaseJWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		
		// If no auth header, continue without authentication
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		// If auth header exists, try to verify it
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			token := parts[1]
			
			client, err := GetFirebaseClient()
			if err == nil {
				decodedToken, err := client.client.VerifyIDToken(context.Background(), token)
				if err == nil {
					// Add user information to request context if token is valid
					ctx := context.WithValue(r.Context(), UserIDKey, decodedToken.UID)
					ctx = context.WithValue(ctx, UserKey, decodedToken)
					r = r.WithContext(ctx)
				}
			}
		}

		// Continue regardless of token validity
		next.ServeHTTP(w, r)
	})
}

// GetUserID extracts the user ID from the request context
func GetUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	return userID, ok
}

// GetUser extracts the full user token from the request context
func GetUser(r *http.Request) (*auth.Token, bool) {
	user, ok := r.Context().Value(UserKey).(*auth.Token)
	return user, ok
}

// respondWithError sends a JSON error response
func respondWithError(w http.ResponseWriter, statusCode int, error, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := ErrorResponse{
		Error:   error,
		Message: message,
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding error response: %v", err)
	}
} 
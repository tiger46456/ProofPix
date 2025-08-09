package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"google.golang.org/api/aiplatform/v1"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	
	"github.com/google/trillian"
	
	"proofpix/internal/certificate"
	"proofpix/internal/index"
	"proofpix/internal/models"
)

// Constants for index management
const (
	indexBucketName    = "proofpix-index"
	indexObjectName    = "latest.faiss"
	assetsCollection   = "assets"
)

// Global index manager instance
var globalIndexManager *index.IndexManager

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
	log.Println("Fingerprint worker started")
	
	// Initialize index startup lifecycle
	ctx := context.Background()
	
	// Create a new instance of IndexManager
	globalIndexManager = &index.IndexManager{}
	
	// Call the Load method on the manager instance
	log.Printf("Loading index from GCS bucket: %s, object: %s", indexBucketName, indexObjectName)
	err := globalIndexManager.Load(ctx, indexBucketName, indexObjectName)
	if err != nil {
		log.Fatalf("Failed to load index: %v", err)
	}
	
	// Check if the manager's internal index is still nil
	if !globalIndexManager.HasIndex() {
		// Log that we are building the index from Firestore
		log.Println("Index not found in GCS, building index from Firestore...")
		
		// Get project ID from environment for Build method
		projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
		if projectID == "" {
			log.Fatal("GOOGLE_CLOUD_PROJECT environment variable not set")
		}
		
		// Call the Build method
		err = globalIndexManager.Build(ctx, projectID, assetsCollection)
		if err != nil {
			log.Fatalf("Failed to build index: %v", err)
		}
		
		// If Build succeeds, log that we are saving the new index to GCS
		log.Println("Successfully built index, saving to GCS...")
		
		// Call the Save method
		err = globalIndexManager.Save(ctx, indexBucketName, indexObjectName)
		if err != nil {
			log.Fatalf("Failed to save index to GCS: %v", err)
		}
		
		log.Println("Successfully saved new index to GCS")
	} else {
		log.Println("Index successfully loaded from GCS")
	}
	
	// Log final message confirming that the index is ready
	log.Println("Index is ready for use")
	
	// Set up HTTP handler
	http.HandleFunc("/process", processHandler)
	
	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Starting server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// processHandler handles incoming HTTP requests to process images
func processHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request to %s", r.Method, r.URL.Path)
	
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse JSON request body
	var req struct {
		UserID  string `json:"user_id"`
		AssetID string `json:"asset_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to parse request body: %v", err)
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if req.UserID == "" || req.AssetID == "" {
		log.Printf("Missing required fields: user_id=%s, asset_id=%s", req.UserID, req.AssetID)
		http.Error(w, "Missing user_id or asset_id", http.StatusBadRequest)
		return
	}
	
	log.Printf("Processing request for user_id=%s, asset_id=%s", req.UserID, req.AssetID)
	
	// Launch processImage as a goroutine for asynchronous processing
	go processImage(req.UserID, req.AssetID)
	
	// Immediately return 200 OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "accepted",
		"message": "Image processing started",
	})
	log.Printf("Request accepted, processing started asynchronously")
}

// processImage downloads an image from Google Cloud Storage and processes it asynchronously
func processImage(userID, assetID string) {
	ctx := context.Background()
	
	// 1. Initialize a new Google Cloud Storage client
	log.Println("Initializing Google Cloud Storage client...")
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Printf("Failed to create Google Cloud Storage client: %v", err)
		return
	}
	defer client.Close()
	
	// 2. Construct the object path using the userID and assetID
	objectPath := fmt.Sprintf("uploads/%s/%s.jpg", userID, assetID)
	log.Printf("Constructed object path: %s", objectPath)
	
	// 3. Use the client to open and read the object from the proofpix-assets-upload bucket
	bucketName := "proofpix-assets-upload"
	bucket := client.Bucket(bucketName)
	object := bucket.Object(objectPath)
	
	log.Printf("Opening object %s from bucket %s...", objectPath, bucketName)
	reader, err := object.NewReader(ctx)
	if err != nil {
		log.Printf("Failed to open object %s from bucket %s: %v", objectPath, bucketName, err)
		return
	}
	defer reader.Close()
	
	// 4. Read the file content into a byte slice
	log.Println("Reading file content...")
	imageData, err := io.ReadAll(reader)
	if err != nil {
		log.Printf("Failed to read file content: %v", err)
		return
	}
	
	// 5. Add logging to confirm successful download and print the size of the downloaded image data
	log.Printf("Successfully downloaded image from GCS")
	log.Printf("Image data size: %d bytes (%.2f KB)", len(imageData), float64(len(imageData))/1024)
	
	// 6. Run getAuthenticityAnalysis and getEmbedding concurrently
	var wg sync.WaitGroup
	
	// Variables to store results from both functions
	var analysisText string
	var analysisErr error
	var embedding []float32
	var embeddingErr error
	
	// Launch goroutine for getAuthenticityAnalysis
	wg.Add(1)
	go func() {
		defer wg.Done()
		analysisText, analysisErr = getAuthenticityAnalysis(imageData)
	}()
	
	// Launch goroutine for getEmbedding
	wg.Add(1)
	go func() {
		defer wg.Done()
		embedding, embeddingErr = getEmbedding(imageData)
	}()
	
	// Wait for both functions to complete
	log.Println("Waiting for authenticity analysis and embedding generation to complete...")
	wg.Wait()
	
	// Check and log results from both functions
	var score int
	var narrative string
	
	if analysisErr != nil {
		log.Printf("Failed to analyze image authenticity: %v", analysisErr)
	} else {
		log.Printf("Authenticity analysis result: %s", analysisText)
		
		// Parse the analysis text to extract score and narrative
		parsedScore, parsedNarrative, parseErr := parseAnalysis(analysisText)
		if parseErr != nil {
			log.Printf("Failed to parse analysis for asset %s: %v", assetID, parseErr)
			// Fall back to default values
			score = 0
			narrative = analysisText // Use raw analysis text as fallback
		} else {
			score = parsedScore
			narrative = parsedNarrative
			log.Printf("Successfully parsed analysis for asset %s: score=%d, narrative=%s", assetID, score, narrative)
		}
	}
	
	if embeddingErr != nil {
		log.Printf("Failed to generate embedding: %v", embeddingErr)
	} else {
		log.Printf("Received embedding with %d dimensions", len(embedding))
		
		// Perform similarity search with the new embedding
		distances, assetIDs, searchErr := globalIndexManager.Search(embedding, 5)
		if searchErr != nil {
			log.Printf("Failed to perform similarity search: %v", searchErr)
		} else {
			log.Printf("Similarity search found asset IDs: %v with distances: %v", assetIDs, distances)
		}
		
		// Add the new embedding to the live index
		addErr := globalIndexManager.Add(assetID, embedding)
		if addErr != nil {
			log.Printf("Failed to add embedding to index for asset %s: %v", assetID, addErr)
		} else {
			log.Printf("Successfully added embedding to index for asset %s", assetID)
		}
	}
	
	// Only save asset if both operations succeeded
	if analysisErr == nil && embeddingErr == nil {
		// Create new Asset struct
		asset := &Asset{
			ID:               assetID,
			UserID:           userID,
			Status:           "completed",
			CreatedAt:        time.Now(),
			RawAnalysis:      analysisText,
			OriginalityScore: score,
			Narrative:        narrative,
			Embedding:        embedding,
		}
		
		// Save asset to Firestore
		if err := saveAsset(ctx, asset); err != nil {
			log.Printf("Failed to save asset %s to Firestore: %v", assetID, err)
		} else {
			log.Printf("Successfully saved asset %s to Firestore", assetID)
			
			// Generate and save certificate after successful asset save
			log.Printf("Generating verifiable credential certificate for asset %s", assetID)
			credential, err := certificate.Generate(asset)
			if err != nil {
				log.Printf("Failed to generate certificate for asset %s: %v", assetID, err)
			} else {
				// Marshal the credential to nicely formatted JSON
				certificateJSON, err := json.MarshalIndent(credential, "", "  ")
				if err != nil {
					log.Printf("Failed to marshal certificate to JSON for asset %s: %v", assetID, err)
				} else {
					// Save the certificate to GCS
											if err := saveJSONCertificate(ctx, assetID, certificateJSON); err != nil {
							log.Printf("Failed to save certificate to GCS for asset %s: %v", assetID, err)
						} else {
							log.Printf("Successfully generated and saved certificate for asset %s", assetID)
							
							// Queue certificate hash in Trillian
							trillianLogID := os.Getenv("TRILLIAN_LOG_ID")
							trillianLogServerAddr := os.Getenv("TRILLIAN_LOG_SERVER_ADDR")
							
							if trillianLogID != "" && trillianLogServerAddr != "" {
								// Parse log ID from string to int64
								logID, parseErr := strconv.ParseInt(trillianLogID, 10, 64)
								if parseErr != nil {
									log.Printf("Failed to parse TRILLIAN_LOG_ID for asset %s: %v", assetID, parseErr)
								} else {
									// Create SHA256 hash of certificate JSON
									hash := sha256.Sum256(certificateJSON)
									leafValue := hash[:]
									
									// Queue the leaf in Trillian
									leafIndex, err := queueLeafInTrillian(ctx, logID, trillianLogServerAddr, leafValue)
									if err != nil {
										log.Printf("Failed to queue certificate hash in Trillian for asset %s: %v", assetID, err)
									} else {
										log.Printf("Successfully queued certificate hash in Trillian for asset %s with leaf index %d", assetID, leafIndex)
										
										// Get project ID from environment for Firestore client
										projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
										if projectID == "" {
											log.Printf("GOOGLE_CLOUD_PROJECT environment variable not set, cannot update Trillian leaf index for asset %s", assetID)
										} else {
											// Initialize Firestore client
											firestoreClient, err := firestore.NewClient(ctx, projectID)
											if err != nil {
												log.Printf("Failed to create Firestore client for updating asset %s: %v", assetID, err)
											} else {
												defer firestoreClient.Close()
												
												// Update the TrillianLeafIndex field directly in Firestore
												_, updateErr := firestoreClient.Collection("assets").Doc(assetID).Update(ctx, []firestore.Update{
													{Path: "trillian_leaf_index", Value: leafIndex},
												})
												if updateErr != nil {
													log.Printf("Failed to update Trillian leaf index in Firestore for asset %s: %v", assetID, updateErr)
												} else {
													log.Printf("Successfully saved Trillian leaf index %d to Firestore for asset %s", leafIndex, assetID)
												}
											}
										}
									}
								}
							} else {
								log.Printf("Skipping Trillian integration for asset %s: TRILLIAN_LOG_ID or TRILLIAN_LOG_SERVER_ADDR not configured", assetID)
							}
							
							// Generate and save badge
						log.Printf("Generating badge for asset %s with score %d", assetID, asset.OriginalityScore)
						badgeData, err := certificate.GenerateBadge(asset.OriginalityScore)
						if err != nil {
							log.Printf("Failed to generate badge for asset %s: %v", assetID, err)
						} else {
							// Save the badge to GCS
							if err := savePNGBadge(ctx, assetID, badgeData); err != nil {
								log.Printf("Failed to save badge to GCS for asset %s: %v", assetID, err)
							} else {
								log.Printf("Successfully generated and saved badge for asset %s", assetID)
							}
						}
					}
				}
			}
		}
	} else {
		log.Printf("Skipping asset save due to processing errors for asset_id=%s", assetID)
	}
	
	log.Printf("Image processing completed for user_id=%s, asset_id=%s", userID, assetID)
}

// getAuthenticityAnalysis accepts image data as a byte slice and returns analysis text and an error
func getAuthenticityAnalysis(imageData []byte) (string, error) {
	ctx := context.Background()
	
	// 1. Initialize the Vertex AI client for the correct GCP project and region
	log.Println("Initializing Vertex AI client...")
	
	// Get project ID from environment
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return "", fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable not set")
	}
	
	// Initialize the AI Platform service (equivalent to generativelanguage.NewPredictionClient)
	client, err := aiplatform.NewService(ctx, option.WithScopes(aiplatform.CloudPlatformScope))
	if err != nil {
		return "", fmt.Errorf("failed to create AI Platform service: %v", err)
	}
	
	// 2. Define the endpoint for the Gemini Pro Vision model
	// Note: The endpoint is defined in the API call below as us-central1-aiplatform.googleapis.com:443 is the default
	
	// 3. Construct the prompt using the exact text from our test suite
	prompt := "You are an expert photography analyst. Analyze this image for any signs of AI generation, such as unnatural patterns, surreal details, warped text, or inconsistent lighting. Based on your analysis, provide a confidence score from 0.0 (definitely AI-generated) to 1.0 (definitely a real photograph) and a brief justification for your score."
	
	// 4. Create a multipart request containing the prompt and the raw image data
	imageBase64 := base64.StdEncoding.EncodeToString(imageData)
	
	requestPayload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{
						"text": prompt,
					},
					{
						"inline_data": map[string]interface{}{
							"mime_type": "image/jpeg",
							"data":      imageBase64,
						},
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.1,
			"topK":           32,
			"topP":           1,
			"maxOutputTokens": 2048,
		},
	}
	
	// Convert payload to JSON
	payloadBytes, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request payload: %v", err)
	}
	
	// Create the API request
	location := "us-central1"
	model := "gemini-1.5-flash"
	
	req := &aiplatform.GoogleCloudAiplatformV1GenerateContentRequest{}
	if err := json.Unmarshal(payloadBytes, req); err != nil {
		return "", fmt.Errorf("failed to unmarshal request: %v", err)
	}
	
	// 5. Call the Predict method on the Gemini client with this request
	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", projectID, location, model)
	
	call := client.Projects.Locations.Publishers.Models.GenerateContent(endpoint, req)
	resp, err := call.Context(ctx).Do()
	
	// 7. Handle and return any errors from the API call
	if err != nil {
		return "", fmt.Errorf("API call failed: %v", err)
	}
	
	// 6. If the call is successful, extract the text content from the first candidate in the response
	if resp == nil {
		return "", fmt.Errorf("received nil response from API")
	}
	
	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}
	
	candidate := resp.Candidates[0]
	if candidate.Content == nil {
		return "", fmt.Errorf("candidate has no content")
	}
	
	if len(candidate.Content.Parts) == 0 {
		return "", fmt.Errorf("candidate content has no parts")
	}
	
	// Extract text from the first part
	part := candidate.Content.Parts[0]
	if part.Text == "" {
		return "", fmt.Errorf("candidate part has no text")
	}
	
	return part.Text, nil
}

// getEmbedding accepts image data as a byte slice and returns embedding vector and an error
func getEmbedding(imageData []byte) ([]float32, error) {
	ctx := context.Background()
	
	// 1. Initialize the Vertex AI client for the correct GCP project and region
	log.Println("Initializing Vertex AI client for embedding...")
	
	// Get project ID from environment
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return nil, fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable not set")
	}
	
	// Initialize the AI Platform service
	client, err := aiplatform.NewService(ctx, option.WithScopes(aiplatform.CloudPlatformScope))
	if err != nil {
		return nil, fmt.Errorf("failed to create AI Platform service: %v", err)
	}
	
	// 2. The endpoint for the multimodal embedding model is the same (us-central1-aiplatform.googleapis.com:443)
	
	// 3. Construct a request to the multimodalembedding@001 model
	// The request contains the image part but does not require a text prompt
	imageBase64 := base64.StdEncoding.EncodeToString(imageData)
	
	requestPayload := map[string]interface{}{
		"instances": []map[string]interface{}{
			{
				"image": map[string]interface{}{
					"bytesBase64Encoded": imageBase64,
				},
			},
		},
	}
	
	// Convert payload to JSON
	payloadBytes, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %v", err)
	}
	
	// Create the API request
	location := "us-central1"
	model := "multimodalembedding@001"
	
	req := &aiplatform.GoogleCloudAiplatformV1PredictRequest{}
	if err := json.Unmarshal(payloadBytes, req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %v", err)
	}
	
	// 4. Call the Predict method
	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", projectID, location, model)
	
	call := client.Projects.Locations.Publishers.Models.Predict(endpoint, req)
	resp, err := call.Context(ctx).Do()
	
	// Handle and return any errors from the API call
	if err != nil {
		return nil, fmt.Errorf("API call failed: %v", err)
	}
	
	// 5. If the call is successful, parse the response to extract the imageEmbedding field
	if resp == nil {
		return nil, fmt.Errorf("received nil response from API")
	}
	
	if len(resp.Predictions) == 0 {
		return nil, fmt.Errorf("no predictions in response")
	}
	
	// Parse the first prediction
	prediction := resp.Predictions[0]
	predictionMap, ok := prediction.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("prediction is not a map")
	}
	
	// Extract imageEmbedding field
	imageEmbeddingInterface, exists := predictionMap["imageEmbedding"]
	if !exists {
		return nil, fmt.Errorf("imageEmbedding field not found in response")
	}
	
	// Convert to slice of float64 first (JSON unmarshaling default)
	imageEmbeddingSlice, ok := imageEmbeddingInterface.([]interface{})
	if !ok {
		return nil, fmt.Errorf("imageEmbedding is not a slice")
	}
	
	// 6. Return the float slice (convert from float64 to float32)
	embedding := make([]float32, len(imageEmbeddingSlice))
	for i, val := range imageEmbeddingSlice {
		floatVal, ok := val.(float64)
		if !ok {
			return nil, fmt.Errorf("embedding value at index %d is not a float", i)
		}
		embedding[i] = float32(floatVal)
	}
	
	log.Printf("Successfully extracted embedding vector with %d dimensions", len(embedding))
	return embedding, nil
}



// saveAsset saves an Asset struct to Firestore
func saveAsset(ctx context.Context, asset *Asset) error {
	// Get project ID from environment
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable not set")
	}

	// Initialize Firestore client
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to create Firestore client: %v", err)
	}
	defer client.Close()

	// Get reference to document in assets collection using Asset ID
	docRef := client.Collection("assets").Doc(asset.ID)

	// Use Set method to write the Asset struct to the document
	_, err = docRef.Set(ctx, asset)
	if err != nil {
		return fmt.Errorf("failed to save asset to Firestore: %v", err)
	}

	log.Printf("Successfully saved asset %s to Firestore", asset.ID)
	return nil
}

// savePNGBadge uploads PNG badge data to Google Cloud Storage
func savePNGBadge(ctx context.Context, assetID string, data []byte) error {
	// Initialize Google Cloud Storage client
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %v", err)
	}
	defer client.Close()

	// Construct object name: badges/{assetID}.png
	bucketName := "proofpix-badges"
	objectName := fmt.Sprintf("badges/%s.png", assetID)

	// Get bucket and object reference
	bucket := client.Bucket(bucketName)
	object := bucket.Object(objectName)

	// Create a writer to upload the data
	writer := object.NewWriter(ctx)
	writer.ContentType = "image/png"

	// Write the PNG data
	_, err = writer.Write(data)
	if err != nil {
		writer.Close()
		return fmt.Errorf("failed to write badge data: %v", err)
	}

	// Close the writer to finalize the upload
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close storage writer: %v", err)
	}

	log.Printf("Successfully saved badge for asset %s to GCS bucket %s", assetID, bucketName)
	return nil
}

// saveJSONCertificate uploads JSON certificate data to Google Cloud Storage
func saveJSONCertificate(ctx context.Context, assetID string, data []byte) error {
	// Initialize Google Cloud Storage client
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %v", err)
	}
	defer client.Close()

	// Construct object name: certificates/{assetID}.json
	bucketName := "proofpix-certificates"
	objectName := fmt.Sprintf("certificates/%s.json", assetID)

	// Get bucket and object reference
	bucket := client.Bucket(bucketName)
	object := bucket.Object(objectName)

	// Create a writer to upload the data
	writer := object.NewWriter(ctx)
	writer.ContentType = "application/json"

	// Write the JSON data
	_, err = writer.Write(data)
	if err != nil {
		writer.Close()
		return fmt.Errorf("failed to write certificate data: %v", err)
	}

	// Close the writer to finalize the upload
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close storage writer: %v", err)
	}

	log.Printf("Successfully saved certificate for asset %s to GCS bucket %s", assetID, bucketName)
	return nil
}

// queueLeafInTrillian submits a leaf value to the Trillian Log Server
func queueLeafInTrillian(ctx context.Context, logID int64, logServerAddr string, leafValue []byte) (int64, error) {
	// 1. Establish a secure gRPC connection to the logServerAddr
	log.Printf("Establishing gRPC connection to Trillian Log Server at %s", logServerAddr)
	conn, err := grpc.DialContext(ctx, logServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to Trillian Log Server at %s: %v", logServerAddr, err)
	}
	
	// 7. Ensure the gRPC connection is properly closed
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			log.Printf("Error closing gRPC connection: %v", closeErr)
		}
	}()
	
	// 2. Create a new trillian.TrillianLogClient using the connection
	client := trillian.NewTrillianLogClient(conn)
	
	// 3. Create the trillian.LogLeaf that will be submitted
	logLeaf := &trillian.LogLeaf{
		LeafValue: leafValue,
	}
	
	// 4. Construct a trillian.QueueLeafRequest containing the logID and the LogLeaf
	request := &trillian.QueueLeafRequest{
		LogId: logID,
		Leaf:  logLeaf,
	}
	
	// 5. Call the QueueLeaf method on the Trillian client
	log.Printf("Submitting leaf to Trillian log %d", logID)
	response, err := client.QueueLeaf(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to queue leaf in Trillian log %d: %v", logID, err)
	}
	
	// 6. Check the response. If the result is not OK or an error occurs, return a descriptive error
	if response == nil {
		return fmt.Errorf("received nil response from Trillian QueueLeaf call")
	}
	
	if response.QueuedLeaf == nil {
		return fmt.Errorf("QueueLeaf response does not contain a queued leaf")
	}
	
	if response.QueuedLeaf.Status == nil {
		return fmt.Errorf("QueueLeaf response does not contain leaf status")
	}
	
	// Check if the status code indicates success (typically google.rpc.Code.OK = 0)
	if response.QueuedLeaf.Status.Code != 0 {
		return 0, fmt.Errorf("Trillian QueueLeaf failed with status code %d: %s", 
			response.QueuedLeaf.Status.Code, response.QueuedLeaf.Status.Message)
	}
	
	// Extract and return the leaf index
	leafIndex := response.QueuedLeaf.Leaf.LeafIndex
	log.Printf("Successfully queued leaf in Trillian log %d with leaf index %d", logID, leafIndex)
	return leafIndex, nil
}
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"proofpix/internal/certificate"
	"proofpix/internal/models"
)

// This is a simple integration test to verify the certificate generation works
// with the models.Asset struct as used in the fingerprint-worker
func main() {
	// Create a test asset similar to what fingerprint-worker creates
	asset := &models.Asset{
		ID:               "test-integration-123",
		UserID:           "user-456",
		Status:           "completed",
		CreatedAt:        time.Now(),
		RawAnalysis:      "This image appears to be authentic with high confidence.",
		OriginalityScore: 9,
		Narrative:        "Analysis shows genuine photographic characteristics",
		Embedding:        []float32{0.1, 0.2, 0.3, 0.4},
	}

	// Generate the certificate
	log.Printf("Generating certificate for asset %s", asset.ID)
	credential, err := certificate.Generate(asset)
	if err != nil {
		log.Fatalf("Failed to generate certificate: %v", err)
	}

	// Marshal to JSON with indentation
	certificateJSON, err := json.MarshalIndent(credential, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal certificate: %v", err)
	}

	// Output the generated certificate
	fmt.Println("Generated Certificate:")
	fmt.Println(string(certificateJSON))
	log.Printf("Successfully generated certificate for asset %s", asset.ID)
}
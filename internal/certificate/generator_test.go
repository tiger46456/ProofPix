package certificate

import (
	"testing"
	"time"

	"proofpix/internal/models"
)

func TestGenerate(t *testing.T) {
	// Create a test asset
	testAsset := &models.Asset{
		ID:               "test-asset-123",
		UserID:           "user-456",
		Status:           "completed",
		CreatedAt:        time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		RawAnalysis:      "This image appears to be authentic with no signs of manipulation.",
		OriginalityScore: 8,
		Narrative:        "High confidence in image authenticity",
		Embedding:        []float32{0.1, 0.2, 0.3},
	}

	// Generate the verifiable credential
	credential, err := Generate(testAsset)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify static values
	expectedContext := []string{"https://www.w3.org/2018/credentials/v1", "https://schema.org"}
	if len(credential.Context) != len(expectedContext) {
		t.Errorf("Context length mismatch: got %d, want %d", len(credential.Context), len(expectedContext))
	}
	for i, ctx := range expectedContext {
		if credential.Context[i] != ctx {
			t.Errorf("Context[%d] = %s, want %s", i, credential.Context[i], ctx)
		}
	}

	if credential.Issuer != "https://proofpix.com" {
		t.Errorf("Issuer = %s, want https://proofpix.com", credential.Issuer)
	}

	if credential.Proof.Type != "DataIntegrityProof" {
		t.Errorf("Proof.Type = %s, want DataIntegrityProof", credential.Proof.Type)
	}

	if credential.Proof.ProofPurpose != "assertionMethod" {
		t.Errorf("Proof.ProofPurpose = %s, want assertionMethod", credential.Proof.ProofPurpose)
	}

	// Verify dynamic values
	expectedSubjectID := "urn:proofpix:asset:test-asset-123"
	if credential.CredentialSubject.ID != expectedSubjectID {
		t.Errorf("CredentialSubject.ID = %s, want %s", credential.CredentialSubject.ID, expectedSubjectID)
	}

	if credential.CredentialSubject.Creator != "user-456" {
		t.Errorf("CredentialSubject.Creator = %s, want user-456", credential.CredentialSubject.Creator)
	}

	if credential.CredentialSubject.AuthenticityRating.RatingValue != 8 {
		t.Errorf("AuthenticityRating.RatingValue = %d, want 8", credential.CredentialSubject.AuthenticityRating.RatingValue)
	}

	if credential.CredentialSubject.AuthenticityNarrative != "High confidence in image authenticity" {
		t.Errorf("AuthenticityNarrative = %s, want 'High confidence in image authenticity'", credential.CredentialSubject.AuthenticityNarrative)
	}

	// Verify proof value is not empty
	if credential.Proof.ProofValue == "" {
		t.Error("Proof.ProofValue should not be empty")
	}

	// Verify dates are in ISO 8601 format
	if _, err := time.Parse(time.RFC3339, credential.IssuanceDate); err != nil {
		t.Errorf("IssuanceDate is not valid ISO 8601: %s", credential.IssuanceDate)
	}

	if _, err := time.Parse(time.RFC3339, credential.Proof.Created); err != nil {
		t.Errorf("Proof.Created is not valid ISO 8601: %s", credential.Proof.Created)
	}
}

func TestGenerateNilAsset(t *testing.T) {
	credential, err := Generate(nil)
	if err == nil {
		t.Error("Generate(nil) should return an error")
	}
	if credential != nil {
		t.Error("Generate(nil) should return nil credential")
	}
}

func TestGenerateWithFallbackNarrative(t *testing.T) {
	// Test asset with empty narrative should fallback to RawAnalysis
	testAsset := &models.Asset{
		ID:               "test-asset-456",
		UserID:           "user-789",
		Status:           "completed",
		CreatedAt:        time.Now(),
		RawAnalysis:      "Fallback analysis text",
		OriginalityScore: 5,
		Narrative:        "", // Empty narrative
		Embedding:        []float32{0.4, 0.5, 0.6},
	}

	credential, err := Generate(testAsset)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if credential.CredentialSubject.AuthenticityNarrative != "Fallback analysis text" {
		t.Errorf("AuthenticityNarrative = %s, want 'Fallback analysis text'", credential.CredentialSubject.AuthenticityNarrative)
	}
}
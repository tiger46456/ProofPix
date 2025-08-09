package certificate

import (
	"crypto/sha256"
	"fmt"
	"time"

	"proofpix/internal/models"
)

// Generate creates a VerifiableCredential from the provided Asset data
func Generate(asset *models.Asset) (*VerifiableCredential, error) {
	if asset == nil {
		return nil, fmt.Errorf("asset cannot be nil")
	}

	// Generate proof value from asset ID and created timestamp
	proofData := asset.ID + asset.CreatedAt.Format(time.RFC3339)
	hash := sha256.Sum256([]byte(proofData))
	proofValue := fmt.Sprintf("%x", hash)

	// Set current time as issuance date and proof creation time
	now := time.Now()
	issuanceDate := now.Format(time.RFC3339)
	proofCreated := now.Format(time.RFC3339)

	// Create credential subject ID based on asset ID
	credentialSubjectID := fmt.Sprintf("urn:proofpix:asset:%s", asset.ID)

	// Set rating value based on originality score (1-10 scale)
	ratingValue := asset.OriginalityScore
	if ratingValue < 1 {
		ratingValue = 1
	} else if ratingValue > 10 {
		ratingValue = 10
	}

	// Use narrative from asset or fallback to raw analysis
	authenticityNarrative := asset.Narrative
	if authenticityNarrative == "" {
		authenticityNarrative = asset.RawAnalysis
	}

	// Create the verifiable credential
	credential := &VerifiableCredential{
		Context: []string{
			"https://www.w3.org/2018/credentials/v1",
			"https://schema.org",
		},
		Type: []string{
			"VerifiableCredential",
			"ProofPixAuthenticityCredential",
		},
		Issuer:       "https://proofpix.com",
		IssuanceDate: issuanceDate,
		CredentialSubject: CredentialSubject{
			ID:      credentialSubjectID,
			Type:    "ImageAuthenticityAssertion",
			Creator: asset.UserID,
			AuthenticityRating: AuthenticityRating{
				Type:        "Rating",
				RatingValue: ratingValue,
				BestRating:  10,
				WorstRating: 1,
			},
			AuthenticityNarrative: authenticityNarrative,
		},
		Proof: Proof{
			Type:         "DataIntegrityProof",
			Created:      proofCreated,
			ProofPurpose: "assertionMethod",
			ProofValue:   proofValue,
		},
	}

	return credential, nil
}
package certificate

// VerifiableCredential represents a W3C Verifiable Credential for image authenticity
type VerifiableCredential struct {
	Context           []string          `json:"@context"`
	Type              []string          `json:"@type"`
	Issuer            string            `json:"issuer"`
	IssuanceDate      string            `json:"issuanceDate"`
	CredentialSubject CredentialSubject `json:"credentialSubject"`
	Proof             Proof             `json:"proof"`
}

// CredentialSubject represents the subject of the verifiable credential
type CredentialSubject struct {
	ID                    string            `json:"id"`
	Type                  string            `json:"type"`
	Creator               string            `json:"creator"`
	AuthenticityRating    AuthenticityRating `json:"authenticityRating"`
	AuthenticityNarrative string            `json:"authenticityNarrative"`
}

// AuthenticityRating represents a schema.org-style rating for image authenticity
type AuthenticityRating struct {
	Type        string `json:"@type"`
	RatingValue int    `json:"ratingValue"`
	BestRating  int    `json:"bestRating"`
	WorstRating int    `json:"worstRating"`
}

// Proof represents cryptographic proof for the verifiable credential
type Proof struct {
	Type         string `json:"type"`
	Created      string `json:"created"`
	ProofPurpose string `json:"proofPurpose"`
	ProofValue   string `json:"proofValue"`
}
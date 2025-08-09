package models

import "time"

// Asset represents a document in Firestore
type Asset struct {
	ID               string    `firestore:"id,omitempty"`
	UserID           string    `firestore:"user_id"`
	Status           string    `firestore:"status"`
	CreatedAt        time.Time `firestore:"created_at"`
	RawAnalysis      string    `firestore:"raw_analysis"`
	OriginalityScore int       `firestore:"originality_score"`
	Narrative        string    `firestore:"narrative"`
	Embedding        []float32 `firestore:"embedding"`
}
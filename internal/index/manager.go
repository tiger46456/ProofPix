package index

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"sync"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"github.com/DataIntelligenceCrew/go-faiss"
	"google.golang.org/api/iterator"
)

// IndexManager manages FAISS indices and provides thread-safe operations
type IndexManager struct {
	index faiss.Index
	idMap map[int64]string
	mu    sync.RWMutex
}

// Load downloads and loads a FAISS index from Google Cloud Storage
func (m *IndexManager) Load(ctx context.Context, bucketName, objectName string) error {
	// Initialize a Google Cloud Storage client
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	// Get the GCS object handle
	obj := client.Bucket(bucketName).Object(objectName)
	
	// Attempt to download the GCS object
	reader, err := obj.NewReader(ctx)
	if err != nil {
		// If the download error is storage.ErrObjectNotExist, log and return nil
		if err == storage.ErrObjectNotExist {
			log.Printf("Index file not found in GCS: gs://%s/%s", bucketName, objectName)
			return nil
		}
		return err
	}
	defer reader.Close()

	// Create a temporary file to store the downloaded index
	tempFile, err := os.CreateTemp("", "faiss_index_*.bin")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name()) // Clean up temporary file
	defer tempFile.Close()

	// Read the object contents into the temporary file
	_, err = io.Copy(tempFile, reader)
	if err != nil {
		return err
	}

	// Close the temp file before reading it with FAISS
	tempFile.Close()

	// Use faiss.ReadIndex to load the index from the temporary file
	loadedIndex, err := faiss.ReadIndex(tempFile.Name())
	if err != nil {
		return err
	}

	// Use mutex lock before writing to m.index
	m.mu.Lock()
	m.index = loadedIndex
	m.mu.Unlock()

	return nil
}

// Build creates a new FAISS index from Firestore documents containing embeddings
func (m *IndexManager) Build(ctx context.Context, projectID, collectionName string) error {
	// Initialize a Firestore client
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return err
	}
	defer client.Close()

	// Query all documents in the specified collection
	iter := client.Collection(collectionName).Documents(ctx)
	defer iter.Stop()

	// Create local slices to hold vectors and asset IDs
	var vectors [][]float32
	var assetIDs []string

	// Iterate through the documents
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		// Unmarshal the document data
		data := doc.Data()
		
		// Check if the document contains an embedding
		if embeddingData, exists := data["embedding"]; exists {
			// Convert embedding to []float32
			if embeddingSlice, ok := embeddingData.([]interface{}); ok {
				vector := make([]float32, len(embeddingSlice))
				for i, val := range embeddingSlice {
					if floatVal, ok := val.(float64); ok {
						vector[i] = float32(floatVal)
					}
				}
				
				// Get the asset ID (use document ID if no specific asset ID field)
				assetID := doc.Ref.ID
				if assetIDData, exists := data["assetId"]; exists {
					if assetIDStr, ok := assetIDData.(string); ok {
						assetID = assetIDStr
					}
				}
				
				// Append to local slices
				vectors = append(vectors, vector)
				assetIDs = append(assetIDs, assetID)
			}
		}
	}

	// Create a new FAISS index with dimension 1408 (Gemini's multimodal embedding dimension)
	index, err := faiss.NewIndexFlatL2(1408)
	if err != nil {
		return err
	}

	// Add all collected vectors to the index
	if len(vectors) > 0 {
		// Convert [][]float32 to the format expected by FAISS
		flatVectors := make([]float32, len(vectors)*1408)
		for i, vector := range vectors {
			copy(flatVectors[i*1408:(i+1)*1408], vector)
		}
		
		err = index.Add(flatVectors)
		if err != nil {
			return err
		}
	}

	// Wrap modifications to m.index and m.idMap in mutex lock
	m.mu.Lock()
	defer m.mu.Unlock()

	// Set the new index
	m.index = index

	// Populate the idMap by mapping index position to asset ID
	m.idMap = make(map[int64]string)
	for i, assetID := range assetIDs {
		m.idMap[int64(i)] = assetID
	}

	return nil
}

// Save uploads the FAISS index to Google Cloud Storage
func (m *IndexManager) Save(ctx context.Context, bucketName, objectName string) error {
	// Check if m.index is nil
	m.mu.RLock()
	if m.index == nil {
		m.mu.RUnlock()
		return errors.New("no index to save: index is nil")
	}
	index := m.index
	m.mu.RUnlock()

	// Create a temporary file on disk
	tempFile, err := os.CreateTemp("", "faiss_index_save_*.bin")
	if err != nil {
		return err
	}
	tempFileName := tempFile.Name()
	defer os.Remove(tempFileName) // Ensure temporary file is removed
	
	// Close the temp file so FAISS can write to it
	tempFile.Close()

	// Use faiss.WriteIndex to save the index to the temporary file
	err = faiss.WriteIndex(index, tempFileName)
	if err != nil {
		return err
	}

	// Reopen the temp file for reading
	tempFile, err = os.Open(tempFileName)
	if err != nil {
		return err
	}
	defer tempFile.Close()

	// Initialize a Google Cloud Storage client
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	// Get the GCS object handle for upload
	obj := client.Bucket(bucketName).Object(objectName)
	
	// Create a writer to upload the file
	writer := obj.NewWriter(ctx)
	defer writer.Close()

	// Copy the temporary file contents to GCS
	_, err = io.Copy(writer, tempFile)
	if err != nil {
		return err
	}

	// Close the writer to finalize the upload
	return writer.Close()
}

// HasIndex returns true if the manager has a loaded index, false otherwise
func (m *IndexManager) HasIndex() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.index != nil
}

// Search performs a similarity search on the index and returns distances and asset IDs
func (m *IndexManager) Search(vector []float32, k int) (distances []float32, assetIDs []string, err error) {
	// Use a read lock at the beginning and defer the unlock
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Check if m.index is nil or has 0 vectors
	if m.index == nil {
		return []float32{}, []string{}, nil
	}
	
	// Check if index has 0 vectors
	if m.index.Ntotal() == 0 {
		return []float32{}, []string{}, nil
	}
	
	// Call the m.index.Search() method, passing the vector and k
	distances, labels, err := m.index.Search(vector, k)
	if err != nil {
		return nil, nil, err
	}
	
	// Create a new slice for the string assetIDs
	assetIDs = make([]string, len(labels))
	
	// Iterate through the integer labels returned by the search
	for i, label := range labels {
		// Look up the corresponding asset ID string from m.idMap
		if assetID, exists := m.idMap[label]; exists {
			assetIDs[i] = assetID
		} else {
			// Handle case where label is not found in idMap
			assetIDs[i] = ""
		}
	}
	
	// Return the final distances and asset IDs
	return distances, assetIDs, nil
}

// Add adds a new vector to the index with the given asset ID
func (m *IndexManager) Add(assetID string, vector []float32) error {
	// Use a write lock at the beginning and defer the unlock
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if m.index is nil
	if m.index == nil {
		return errors.New("index is not initialized")
	}

	// Get the current total number of items in the index
	newID := m.index.Ntotal()

	// Call m.index.Add() with a slice containing just the new vector
	err := m.index.Add(vector)
	if err != nil {
		return err
	}

	// After a successful add, update the m.idMap
	if m.idMap == nil {
		m.idMap = make(map[int64]string)
	}
	m.idMap[newID] = assetID

	return nil
}
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"google.golang.org/api/aiplatform/v1"
	"google.golang.org/api/option"
)

// ImageResult represents the analysis result for a single image
type ImageResult struct {
	Filename        string
	KnownType       string // "real" or "ai"
	ConfidenceScore float64
	Justification   string
	Error           string
}

// GeminiResponse represents the response structure from Gemini API
type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

const prompt = "You are an expert photography analyst. Analyze this image for any signs of AI generation, such as unnatural patterns, surreal details, warped text, or inconsistent lighting. Based on your analysis, provide a confidence score from 0.0 (definitely AI-generated) to 1.0 (definitely a real photograph) and a brief justification for your score."

func main() {
	fmt.Println("ProofPix Image Analysis Test Suite")
	fmt.Println("==================================")

	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// Define test image directories
	realDir := filepath.Join(wd, "cmd", "test-suite", "test-images", "real")
	aiDir := filepath.Join(wd, "cmd", "test-suite", "test-images", "ai")

	// Initialize Gemini API client
	ctx := context.Background()
	client, err := initGeminiClient(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize Gemini client: %v", err)
	}

	var results []ImageResult

	// Process real images
	fmt.Println("\nProcessing real images...")
	realResults, err := processImagesInDirectory(ctx, client, realDir, "real")
	if err != nil {
		log.Printf("Error processing real images: %v", err)
	}
	results = append(results, realResults...)

	// Process AI images
	fmt.Println("\nProcessing AI images...")
	aiResults, err := processImagesInDirectory(ctx, client, aiDir, "ai")
	if err != nil {
		log.Printf("Error processing AI images: %v", err)
	}
	results = append(results, aiResults...)

	// Print results
	printResults(results)
}

func initGeminiClient(ctx context.Context) (*aiplatform.Service, error) {
	// Check for required environment variables
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return nil, fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable not set")
	}

	// Initialize the AI Platform service
	service, err := aiplatform.NewService(ctx, option.WithScopes(aiplatform.CloudPlatformScope))
	if err != nil {
		return nil, fmt.Errorf("failed to create AI Platform service: %v", err)
	}

	return service, nil
}

func processImagesInDirectory(ctx context.Context, client *aiplatform.Service, dirPath, imageType string) ([]ImageResult, error) {
	var results []ImageResult

	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		fmt.Printf("Directory %s does not exist, skipping...\n", dirPath)
		return results, nil
	}

	// Read directory contents
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %v", dirPath, err)
	}

	if len(files) == 0 {
		fmt.Printf("No files found in %s\n", dirPath)
		return results, nil
	}

	// Process each image file
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Check if file is an image (basic check by extension)
		filename := file.Name()
		if !isImageFile(filename) {
			fmt.Printf("Skipping non-image file: %s\n", filename)
			continue
		}

		fmt.Printf("Processing: %s\n", filename)

		result := ImageResult{
			Filename:  filename,
			KnownType: imageType,
		}

		// Analyze image with Gemini
		filePath := filepath.Join(dirPath, filename)
		score, justification, err := analyzeImageWithGemini(ctx, client, filePath)
		if err != nil {
			result.Error = err.Error()
			log.Printf("Error analyzing %s: %v", filename, err)
		} else {
			result.ConfidenceScore = score
			result.Justification = justification
		}

		results = append(results, result)
	}

	return results, nil
}

func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".tiff", ".tif"}
	
	for _, validExt := range imageExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

func analyzeImageWithGemini(ctx context.Context, client *aiplatform.Service, imagePath string) (float64, string, error) {
	// Read and encode image
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return 0, "", fmt.Errorf("failed to read image file: %v", err)
	}

	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	// Get project ID from environment
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return 0, "", fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable not set")
	}

	// Prepare the request payload for Gemini
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
		return 0, "", fmt.Errorf("failed to marshal request payload: %v", err)
	}

	// Create the API request
	// Note: This uses a simplified approach. In production, you'd want to use the proper Gemini API endpoint
	location := "us-central1"
	model := "gemini-1.5-flash"
	
	req := &aiplatform.GoogleCloudAiplatformV1GenerateContentRequest{}
	if err := json.Unmarshal(payloadBytes, req); err != nil {
		return 0, "", fmt.Errorf("failed to unmarshal request: %v", err)
	}

	// Make the API call
	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", projectID, location, model)
	
	call := client.Projects.Locations.Publishers.Models.GenerateContent(endpoint, req)
	resp, err := call.Context(ctx).Do()
	if err != nil {
		return 0, "", fmt.Errorf("API call failed: %v", err)
	}

	// Parse response
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return 0, "", fmt.Errorf("no response content received")
	}

	responseText := resp.Candidates[0].Content.Parts[0].Text
	score, justification := parseGeminiResponse(responseText)

	return score, justification, nil
}

func parseGeminiResponse(responseText string) (float64, string) {
	// Try to extract confidence score using regex
	scoreRegex := regexp.MustCompile(`(?i)(?:confidence|score)[\s:]*([0-9]*\.?[0-9]+)`)
	matches := scoreRegex.FindStringSubmatch(responseText)
	
	var score float64 = -1 // Default to -1 if no score found
	if len(matches) > 1 {
		if parsedScore, err := strconv.ParseFloat(matches[1], 64); err == nil {
			score = parsedScore
		}
	}

	// Return the full response as justification
	return score, responseText
}

func printResults(results []ImageResult) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("TEST RESULTS")
	fmt.Println(strings.Repeat("=", 80))

	if len(results) == 0 {
		fmt.Println("No images were processed.")
		return
	}

	for i, result := range results {
		fmt.Printf("\n[%d] %s\n", i+1, result.Filename)
		fmt.Printf("Known Type: %s\n", strings.ToUpper(result.KnownType))
		
		if result.Error != "" {
			fmt.Printf("ERROR: %s\n", result.Error)
		} else {
			if result.ConfidenceScore >= 0 {
				fmt.Printf("Confidence Score: %.2f\n", result.ConfidenceScore)
			} else {
				fmt.Printf("Confidence Score: Could not parse from response\n")
			}
			fmt.Printf("Justification: %s\n", result.Justification)
		}
		
		fmt.Println(strings.Repeat("-", 40))
	}

	// Print summary
	fmt.Printf("\nSUMMARY: Processed %d images\n", len(results))
	
	successCount := 0
	for _, result := range results {
		if result.Error == "" {
			successCount++
		}
	}
	
	fmt.Printf("Successful analyses: %d/%d\n", successCount, len(results))
	if successCount < len(results) {
		fmt.Printf("Failed analyses: %d/%d\n", len(results)-successCount, len(results))
	}
}
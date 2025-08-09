package main

import (
	"fmt"
	"regexp"
	"strconv"
)

// parseAnalysis extracts confidence score and justification from raw analysis text
func parseAnalysis(rawText string) (score int, narrative string, err error) {
	// Regular expression to find confidence score (e.g., "Confidence Score: 0.98")
	scoreRegex := regexp.MustCompile(`(?i)confidence\s+score:\s*([0-9]*\.?[0-9]+)`)
	scoreMatch := scoreRegex.FindStringSubmatch(rawText)
	
	if len(scoreMatch) < 2 {
		return 0, "", fmt.Errorf("confidence score not found in raw text")
	}
	
	// Parse the float score
	floatScore, err := strconv.ParseFloat(scoreMatch[1], 64)
	if err != nil {
		return 0, "", fmt.Errorf("failed to parse confidence score: %v", err)
	}
	
	// Convert to integer percentage (0.98 -> 98)
	score = int(floatScore * 100)
	
	// Regular expression to find justification text (e.g., "Justification: ...")
	narrativeRegex := regexp.MustCompile(`(?i)justification:\s*(.+?)(?:\n\n|\z)`)
	narrativeMatch := narrativeRegex.FindStringSubmatch(rawText)
	
	if len(narrativeMatch) < 2 {
		return 0, "", fmt.Errorf("justification text not found in raw text")
	}
	
	// Extract and clean the narrative text
	narrative = narrativeMatch[1]
	
	return score, narrative, nil
}
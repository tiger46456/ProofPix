package main

import (
	"testing"
)

func TestParseAnalysis_Success(t *testing.T) {
	// Define a sample input string that mimics a perfect response from Gemini
	input := "Confidence Score: 0.98\n\nJustification: The lighting and shadows appear natural."
	
	// Call the parseAnalysis function from analysis.go with this sample input
	score, narrative, err := parseAnalysis(input)
	
	// Use assertions to check the results
	if score != 98 {
		t.Errorf("Expected score to be 98, but got %d", score)
	}
	
	if narrative != "The lighting and shadows appear natural." {
		t.Errorf("Expected narrative to be 'The lighting and shadows appear natural.', but got '%s'", narrative)
	}
	
	if err != nil {
		t.Errorf("Expected err to be nil, but got %v", err)
	}
}

func TestParseAnalysis_EdgeCases(t *testing.T) {
	// Table-driven test structure
	testCases := []struct {
		name          string
		input         string
		expectedScore int
		expectError   bool
	}{
		{
			name:          "Missing Confidence Score line",
			input:         "This is some analysis text.\n\nJustification: The image looks authentic.",
			expectedScore: 0,
			expectError:   true,
		},
		{
			name:          "Missing Justification line",
			input:         "Confidence Score: 0.85\n\nThis is some other text without justification.",
			expectedScore: 0,
			expectError:   true,
		},
		{
			name:          "Completely empty input",
			input:         "",
			expectedScore: 0,
			expectError:   true,
		},
		{
			name:          "Malformed gibberish input",
			input:         "Random gibberish text 12345 !@#$% no structure at all",
			expectedScore: 0,
			expectError:   true,
		},
	}

	// Loop through test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call parseAnalysis with the test input
			score, narrative, err := parseAnalysis(tc.input)
			
			// Check if error expectation matches
			if tc.expectError && err == nil {
				t.Errorf("Expected an error for case '%s', but got nil", tc.name)
			}
			
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error for case '%s', but got: %v", tc.name, err)
			}
			
			// Check score (should be 0 for error cases)
			if score != tc.expectedScore {
				t.Errorf("Expected score %d for case '%s', but got %d", tc.expectedScore, tc.name, score)
			}
			
			// For error cases, narrative should be empty
			if tc.expectError && narrative != "" {
				t.Errorf("Expected empty narrative for error case '%s', but got '%s'", tc.name, narrative)
			}
		})
	}
}
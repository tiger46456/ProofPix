package certificate

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
)

// GenerateBadge creates a PNG badge with an authenticity score
// The badge color changes based on the score: green (>=90), orange (>=70), red (<70)
func GenerateBadge(score int) ([]byte, error) {
	// Define badge dimensions
	const (
		width  = 250.0
		height = 60.0
	)

	// Choose background color based on score
	var bgColor color.RGBA
	switch {
	case score >= 90:
		bgColor = color.RGBA{76, 175, 80, 255} // Green
	case score >= 70:
		bgColor = color.RGBA{255, 152, 0, 255} // Orange
	default:
		bgColor = color.RGBA{244, 67, 54, 255} // Red
	}

	// Create a new canvas
	c := canvas.New(width, height)

	// Create background rectangle path and style
	rect := canvas.Rectangle(width, height)
	style := canvas.Style{
		Fill: canvas.Paint{Color: bgColor},
	}
	
	// Add background rectangle to canvas
	c.RenderPath(rect, style, canvas.Identity)

	// Load font face
	// NOTE: This font path must be included in the final Docker image
	// Consider copying DejaVuSans.ttf to /app/fonts/ in the Docker container
	fontPath := "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"
	
	// Try to load the font, fallback to built-in options if not available
	fontFamily := canvas.NewFontFamily("dejavu")
	err := fontFamily.LoadFontFile(fontPath, canvas.FontRegular)
	if err != nil {
		// If external font loading fails, create a minimal font family
		// In production, ensure the font file is available in the Docker image
		fontFamily = canvas.NewFontFamily("fallback")
		
		// Try to load a basic system font
		systemFonts := []string{"Arial", "Times New Roman", "Helvetica", "sans-serif"}
		fontLoaded := false
		
		for _, fontName := range systemFonts {
			err = fontFamily.LoadSystemFont(fontName, canvas.FontRegular)
			if err == nil {
				fontLoaded = true
				break
			}
		}
		
		// If no system fonts work, the canvas will have to work without custom fonts
		// This is a graceful degradation approach
		if !fontLoaded {
			// Use a simple approach: render a badge without text if fonts fail completely
			// In a real production environment, you'd want to bundle fonts with the application
			return nil, fmt.Errorf("unable to load any font for badge generation - please ensure fonts are available in the system or Docker image")
		}
	}

	white := color.RGBA{255, 255, 255, 255}
	face := fontFamily.Face(12.0, white) // White text

	// Add "Authenticity Score" text
	titleText := canvas.NewTextLine(face, "Authenticity Score", canvas.Left)
	titleBounds := titleText.Bounds()
	titleX := (width - titleBounds.W()) / 2 // Center horizontally
	titleY := height - 15.0                 // Position near top
	titleMatrix := canvas.Identity.Translate(titleX, titleY)
	c.RenderText(titleText, titleMatrix)

	// Add score percentage text
	scoreFace := fontFamily.Face(16.0, white) // White text
	scoreText := canvas.NewTextLine(scoreFace, fmt.Sprintf("%d%%", score), canvas.Left)
	scoreBounds := scoreText.Bounds()
	scoreX := (width - scoreBounds.W()) / 2 // Center horizontally
	scoreY := 20.0                          // Position near bottom
	scoreMatrix := canvas.Identity.Translate(scoreX, scoreY)
	c.RenderText(scoreText, scoreMatrix)

	// Render canvas to PNG in memory using the rasterizer
	var buf bytes.Buffer
	ras := rasterizer.New(width, height, canvas.DPMM(3.0), canvas.DefaultColorSpace)
	
	// Render the canvas to the rasterizer, then get the image
	c.RenderTo(ras)
	img := ras.Image
	
	// Encode as PNG using standard library
	err = png.Encode(&buf, img)
	if err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), nil
}
// Marshal Guard screenshot segmentation tool
// Processes WhatsApp images
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg" // Register JPEG decoder
	"image/png"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: mg_segment <input_dir> [output_dir]")
		fmt.Println("Example: mg_segment C:\\Users\\verve\\Downloads F:\\Projects\\LastWar\\mg_output")
		os.Exit(1)
	}

	inputDir := os.Args[1]
	outputDir := "F:\\Projects\\LastWar\\mg_output"
	if len(os.Args) > 2 {
		outputDir = os.Args[2]
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Find all WhatsApp image files only
	files, err := filepath.Glob(filepath.Join(inputDir, "WhatsApp*"))
	if err != nil {
		fmt.Printf("Error reading input directory: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Printf("No WhatsApp images found in %s\n", inputDir)
		os.Exit(1)
	}

	fmt.Printf("Found %d WhatsApp images\n", len(files))

	processedCount := 0
	for _, file := range files {
		ext := filepath.Ext(file)
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			continue
		}

		fmt.Printf("\n=== Processing: %s ===\n", filepath.Base(file))
		if err := processImage(file, outputDir); err != nil {
			fmt.Printf("Error processing %s: %v\n", file, err)
			continue
		}
		processedCount++
	}

	fmt.Printf("\n✓ Processed %d images\n", processedCount)
}

func processImage(imagePath, outputDir string) error {
	// Open and decode image
	f, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	fmt.Printf("Image size: %d × %d\n", width, height)

	// Detect colored rectangles
	fmt.Printf("Detecting colored regions...\n")
	rectangles := findColoredRegions(img, width, height)
	fmt.Printf("Detected %d rectangles\n", len(rectangles))

	// Draw annotations
	annotated := image.NewRGBA(bounds)
	draw.Draw(annotated, bounds, img, bounds.Min, draw.Src)

	colors := []color.RGBA{
		{255, 0, 0, 255},   // Red
		{0, 255, 0, 255},   // Green
		{0, 0, 255, 255},   // Blue
		{255, 255, 0, 255}, // Yellow
		{255, 0, 255, 255}, // Magenta
		{0, 255, 255, 255}, // Cyan
	}

	for i, rect := range rectangles {
		c := colors[i%len(colors)]
		drawRectangle(annotated, rect.x0, rect.y0, rect.x1, rect.y1, c)
	}

	// Save output image
	baseName := filepath.Base(imagePath)
	nameNoExt := baseName[:len(baseName)-len(filepath.Ext(baseName))]
	outputPath := filepath.Join(outputDir, nameNoExt+"_annotated.png")

	if err := saveImage(annotated, outputPath); err != nil {
		return err
	}

	fmt.Printf("✓ Saved: %s\n", filepath.Base(outputPath))
	return nil
}

// Rectangle represents a detected rectangular region
type Rectangle struct {
	x0, y0, x1, y1 int
}

// findColoredRegions detects rectangular regions with uniform non-white color
func findColoredRegions(img image.Image, width, height int) []Rectangle {
	rectangles := []Rectangle{}

	// Scan for horizontal bands with non-white background
	inRegion := false
	regionStart := 0

	for y := 0; y < height; y++ {
		isNonWhiteRow := isRowNonWhite(img, y, width)

		if isNonWhiteRow && !inRegion {
			// Start of a new region
			regionStart = y
			inRegion = true
		} else if !isNonWhiteRow && inRegion {
			// End of region
			if y-regionStart > 20 { // Minimum height threshold
				rectangles = append(rectangles, Rectangle{0, regionStart, width, y})
				fmt.Printf("  Found rectangle: y=%d to %d (height=%d)\n", regionStart, y, y-regionStart)
			}
			inRegion = false
		}
	}

	// Handle case where region extends to bottom
	if inRegion && height-regionStart > 20 {
		rectangles = append(rectangles, Rectangle{0, regionStart, width, height})
		fmt.Printf("  Found rectangle: y=%d to %d (height=%d)\n", regionStart, height, height-regionStart)
	}

	return rectangles
}

// isRowNonWhite checks if a row has predominantly non-white pixels
func isRowNonWhite(img image.Image, y, width int) bool {
	whiteThreshold := uint8(240) // Pixels brighter than this are considered white
	nonWhiteCount := 0
	sampleStep := 5 // Sample every 5th pixel for performance

	for x := 0; x < width; x += sampleStep {
		r, g, b, _ := img.At(x, y).RGBA()
		// Convert to 8-bit
		r8 := uint8(r >> 8)
		g8 := uint8(g >> 8)
		b8 := uint8(b >> 8)

		// Check if pixel is non-white (any channel below threshold)
		if r8 < whiteThreshold || g8 < whiteThreshold || b8 < whiteThreshold {
			nonWhiteCount++
		}
	}

	// If more than 30% of sampled pixels are non-white, consider the row non-white
	sampledPixels := width / sampleStep
	return float64(nonWhiteCount)/float64(sampledPixels) > 0.3
}

// drawRectangle draws a rectangle border
func drawRectangle(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	thickness := 3

	// Top and bottom edges
	for dy := 0; dy < thickness; dy++ {
		drawHorizontalLineSegment(img, y0+dy, x0, x1, c)
		drawHorizontalLineSegment(img, y1-dy-1, x0, x1, c)
	}

	// Left and right edges
	for dx := 0; dx < thickness; dx++ {
		drawVerticalLine(img, x0+dx, y0, y1, c)
		drawVerticalLine(img, x1-dx-1, y0, y1, c)
	}
}

// drawHorizontalLineSegment draws a horizontal line segment from x0 to x1
func drawHorizontalLineSegment(img *image.RGBA, y, x0, x1 int, c color.Color) {
	bounds := img.Bounds()
	if y < bounds.Min.Y || y >= bounds.Max.Y {
		return
	}

	if x0 < bounds.Min.X {
		x0 = bounds.Min.X
	}
	if x1 > bounds.Max.X {
		x1 = bounds.Max.X
	}

	for x := x0; x < x1; x++ {
		img.Set(x, y, c)
	}
}

// drawVerticalLine draws a vertical line from y0 to y1
func drawVerticalLine(img *image.RGBA, x, y0, y1 int, c color.Color) {
	bounds := img.Bounds()
	if x < bounds.Min.X || x >= bounds.Max.X {
		return
	}

	if y0 < bounds.Min.Y {
		y0 = bounds.Min.Y
	}
	if y1 > bounds.Max.Y {
		y1 = bounds.Max.Y
	}

	for y := y0; y < y1; y++ {
		img.Set(x, y, c)
	}
}

// saveImage writes an image to a PNG file
func saveImage(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

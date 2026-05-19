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
	"sort"
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

	// Step 1: Find the dialog (big white rectangle) using color detection
	dialogX0, dialogY0, dialogX1, dialogY1 := findDialogBounds(img, width, height)
	fmt.Printf("Dialog bounds: x=%d-%d y=%d-%d\n", dialogX0, dialogX1, dialogY0, dialogY1)

	// Step 2: Crop image to dialog bounds
	cropW := dialogX1 - dialogX0
	cropH := dialogY1 - dialogY0
	cropped := image.NewRGBA(image.Rect(0, 0, cropW, cropH))
	draw.Draw(cropped, cropped.Bounds(), img, image.Pt(dialogX0, dialogY0), draw.Src)

	// Step 3: Detect colored sections within the cropped dialog image
	fmt.Printf("Detecting colored regions...\n")
	rectangles := findColoredRegions(cropped, cropW, cropH)
	fmt.Printf("Detected %d rectangles\n", len(rectangles))

	// Step 4: Draw outer dialog annotation — orange border at the crop boundary
	drawRectangle(cropped, 0, 0, cropW, cropH, color.RGBA{255, 165, 0, 255})

	// Step 5: Draw inner section annotations + sub-segments
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
		drawRectangle(cropped, rect.x0, rect.y0, rect.x1, rect.y1, c)

		// Split rectangle into segments and annotate each
		segments := splitRectangle(cropped, rect)
		fmt.Printf("  rect %d → %d segments\n", i+1, len(segments))
		for _, seg := range segments {
			// Outer annotation in section color
			drawRectangle(cropped, seg.x0, seg.y0, seg.x1, seg.y1, c)
		}
	}

	// Save cropped annotated image
	baseName := filepath.Base(imagePath)
	nameNoExt := baseName[:len(baseName)-len(filepath.Ext(baseName))]
	outputPath := filepath.Join(outputDir, nameNoExt+"_annotated.png")

	if err := saveImage(cropped, outputPath); err != nil {
		return err
	}

	fmt.Printf("✓ Saved: %s\n", filepath.Base(outputPath))
	return nil
}

// findDialogBounds detects the bounding box of the main dialog popup.
// The dialog sits inside a dark navy outer frame and a dark game header/footer.
// For x bounds: scans the full image inward from each edge within the middle third
// of the image height, looking for where brightness rises above 170 (leaving dark frame).
// For y bounds: scans within that x strip for where brightness rises above 200.
func findDialogBounds(img image.Image, width, height int) (x0, y0, x1, y1 int) {
	// x bounds: build column means across the middle third of the image
	sampleY0 := height / 3
	sampleY1 := height * 2 / 3
	colMean := make([]int, width)
	for x := 0; x < width; x++ {
		sum, count := 0, 0
		for y := sampleY0; y < sampleY1; y += 2 {
			r, g, b, _ := img.At(x, y).RGBA()
			sum += (int(r>>8) + int(g>>8) + int(b>>8)) / 3
			count++
		}
		if count > 0 {
			colMean[x] = sum / count
		}
	}
	// Scan from left: first column where brightness rises above 170 (end of dark frame)
	x0 = 0
	for x := 0; x < width; x++ {
		if colMean[x] > 170 {
			x0 = x
			break
		}
	}
	// Scan from right: last column where brightness > 170
	x1 = width
	for x := width - 1; x >= 0; x-- {
		if colMean[x] > 170 {
			x1 = x + 1
			break
		}
	}

	// y bounds: within the x0..x1 strip, find where brightness rises above 200
	lightThreshold := 200

	// Top: scan downward — find first light row
	y0 = 0
	for y := 0; y < height; y++ {
		if rowMeanInRange(img, y, x0, x1) > lightThreshold {
			y0 = y
			break
		}
	}

	// Bottom: scan upward — find last light row
	y1 = height
	for y := height - 1; y >= 0; y-- {
		if rowMeanInRange(img, y, x0, x1) > lightThreshold {
			y1 = y + 1
			break
		}
	}

	return x0, y0, x1, y1
}

// rowMeanInRange returns the mean brightness of a row between x0 and x1.
func rowMeanInRange(img image.Image, y, x0, x1 int) int {
	sum, count := 0, 0
	for x := x0; x < x1; x += 5 {
		r, g, b, _ := img.At(x, y).RGBA()
		sum += (int(r>>8) + int(g>>8) + int(b>>8)) / 3
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / count
}

// Rectangle represents a detected rectangular region
type Rectangle struct {
	x0, y0, x1, y1 int
}

// Segment represents a horizontal sub-segment within a rectangle
type Segment struct {
	x0, y0, x1, y1 int
}

// splitRectangle splits a rectangle into horizontal sub-segments separated by
// runs of 20+ columns whose brightness matches the rectangle's background color.
// Background is detected as the median column brightness across the rectangle.
func splitRectangle(img image.Image, rect Rectangle) []Segment {
	rectW := rect.x1 - rect.x0
	if rectW < 1 {
		return []Segment{{rect.x0, rect.y0, rect.x1, rect.y1}}
	}

	// Compute mean brightness for each column within the rectangle's y band
	colMeans := make([]int, rectW)
	for i := 0; i < rectW; i++ {
		x := rect.x0 + i
		sum, count := 0, 0
		for y := rect.y0; y < rect.y1; y++ {
			r, g, b, _ := img.At(x, y).RGBA()
			sum += (int(r>>8) + int(g>>8) + int(b>>8)) / 3
			count++
		}
		if count > 0 {
			colMeans[i] = sum / count
		}
	}

	// Determine background brightness as the median column mean
	sortedMeans := make([]int, rectW)
	copy(sortedMeans, colMeans)
	sort.Ints(sortedMeans)
	bgMean := sortedMeans[len(sortedMeans)/2]

	// A column is "background" if its mean is within ±25 of the background mean
	bgTolerance := 25
	isBackground := make([]bool, rectW)
	for i, m := range colMeans {
		diff := m - bgMean
		if diff < 0 {
			diff = -diff
		}
		isBackground[i] = diff <= bgTolerance
	}

	// Find segments: content spans separated by runs of 20+ background columns
	const minSeparatorWidth = 15
	segments := []Segment{}

	i := 0
	for i < rectW {
		// Skip leading background columns
		if isBackground[i] {
			i++
			continue
		}

		// Found start of content
		contentStart := i
		contentEnd := i + 1
		i++

		for i < rectW {
			if !isBackground[i] {
				contentEnd = i + 1
				i++
			} else {
				// Measure this background run
				runStart := i
				for i < rectW && isBackground[i] {
					i++
				}
				runLen := i - runStart
				if runLen >= minSeparatorWidth {
					// Wide gap — separator, end this segment
					break
				}
				// Narrow gap — bridge over it
				contentEnd = i
			}
		}

		if contentEnd > contentStart {
			segments = append(segments, Segment{
				rect.x0 + contentStart, rect.y0,
				rect.x0 + contentEnd, rect.y1,
			})
		}
	}

	// Fallback: if nothing split, return the whole rectangle as one segment
	if len(segments) == 0 {
		return []Segment{{rect.x0, rect.y0, rect.x1, rect.y1}}
	}
	return segments
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
				// Find left/right boundaries by scanning columns across the whole band
				x0, x1 := findColumnBounds(img, regionStart, y, width)
				rectangles = append(rectangles, Rectangle{x0, regionStart, x1, y})
				fmt.Printf("  Found rectangle: x=%d-%d y=%d-%d\n", x0, x1, regionStart, y)
			}
			inRegion = false
		}
	}

	// Handle case where region extends to bottom
	if inRegion && height-regionStart > 20 {
		x0, x1 := findColumnBounds(img, regionStart, height, width)
		rectangles = append(rectangles, Rectangle{x0, regionStart, x1, height})
		fmt.Printf("  Found rectangle: x=%d-%d y=%d-%d\n", x0, x1, regionStart, height)
	}

	return rectangles
}

// findColumnBounds finds where colored content starts and ends within a horizontal band.
// The dialog background is near-white (mean ~248). Colored row content (grey-blue headers,
// member card backgrounds) has mean < 230. We scan from each edge inward — skipping the
// single-pixel grey border artefacts at x=0 and x=width-1 — to find the first and last
// column whose band-averaged brightness is below the color threshold.
func findColumnBounds(img image.Image, y0, y1, width int) (x0, x1 int) {
	bandHeight := y1 - y0
	if bandHeight < 1 {
		return 0, width
	}

	// Compute mean brightness per column across the band (sample every other row)
	colMean := make([]int, width)
	for x := 0; x < width; x++ {
		sum, count := 0, 0
		for y := y0; y < y1; y += 2 {
			r, g, b, _ := img.At(x, y).RGBA()
			sum += (int(r>>8) + int(g>>8) + int(b>>8)) / 3
			count++
		}
		if count > 0 {
			colMean[x] = sum / count
		}
	}

	// White dialog background: mean > 230. Colored content: mean < 230.
	// Start 3px in from each edge to skip single-pixel border artefacts.
	colorThreshold := 230
	margin := 3

	leftMost := margin
	for x := margin; x < width-margin; x++ {
		if colMean[x] < colorThreshold {
			leftMost = x
			break
		}
	}

	rightMost := width - margin - 1
	for x := width - margin - 1; x >= margin; x-- {
		if colMean[x] < colorThreshold {
			rightMost = x
			break
		}
	}

	return leftMost, rightMost + 1
}

// findHorizontalBounds scans a row to find the leftmost and rightmost non-white pixels
func findHorizontalBounds(img image.Image, y, width int) (x0, x1 int) {
	whiteThreshold := uint8(240)

	x0 = width
	x1 = 0

	for x := 0; x < width; x++ {
		r, g, b, _ := img.At(x, y).RGBA()
		r8 := uint8(r >> 8)
		g8 := uint8(g >> 8)
		b8 := uint8(b >> 8)

		if r8 < whiteThreshold || g8 < whiteThreshold || b8 < whiteThreshold {
			if x < x0 {
				x0 = x
			}
			if x > x1 {
				x1 = x
			}
		}
	}

	// If nothing found, fall back to full width
	if x0 > x1 {
		return 0, width
	}

	// Add small padding
	if x0 > 2 {
		x0 -= 2
	}
	if x1 < width-2 {
		x1 += 2
	}

	return x0, x1
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

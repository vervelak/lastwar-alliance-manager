// Marshal Guard screenshot segmentation tool
// Processes WhatsApp images
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg" // Register JPEG decoder
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	gosseract "github.com/otiai10/gosseract/v2"
)

var (
	rankRe   = regexp.MustCompile(`^([2-9]|[1-9][0-9]|100)$`)
	nameRe   = regexp.MustCompile(`^\[[A-Za-z0-9]{1,4}\]\s*([A-Za-z0-9 ]+)$`)
	damageRe = regexp.MustCompile(`^Total Damage:\s\d+(?:\.\d{1,2})?[GM]$`)
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

	// Base name used for both the annotated image and the crop subdirectory
	baseName := filepath.Base(imagePath)
	nameNoExt := baseName[:len(baseName)-len(filepath.Ext(baseName))]
	cropDir := filepath.Join(outputDir, nameNoExt)

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
		vertSegs := drawVerticalSegments(cropped, rect, c)
		hSegs := make([][]Rectangle, len(vertSegs))
		for si, seg := range vertSegs {
			hSegs[si] = drawHorizontalSegments(cropped, seg, c)
		}

		// Match the member-row pattern:
		//   5 vertical segments (d-bg-d-bg-d-bg-d-bg-d)
		//   vseg[1]: 3 horizontal regions (d-bg-d-bg-d) → fill middle  [1]
		//   vseg[3]: 4 horizontal regions (d-bg-d-bg-d-bg-d) → fill [1] and [2]
		hCounts := make([]int, len(hSegs))
		for k, hs := range hSegs {
			hCounts[k] = len(hs)
		}
		fmt.Printf("  rect %d: %d vsegs, hsegs=%v\n", i, len(vertSegs), hCounts)
		if len(vertSegs) == 5 && len(hSegs[1]) == 3 && len(hSegs[3]) == 4 {
			rectDir := filepath.Join(cropDir, fmt.Sprintf("rect%02d", i))
			if err := os.MkdirAll(rectDir, 0o755); err != nil {
				return fmt.Errorf("mkdir %s: %w", rectDir, err)
			}
			if err := processMatchedSegment(cropped, hSegs[1][1], filepath.Join(rectDir, "seg1_mid.png"), true); err != nil {
				return err
			}
			if err := processMatchedSegment(cropped, hSegs[3][1], filepath.Join(rectDir, "seg3_2nd.png"), false); err != nil {
				return err
			}
			if err := processMatchedSegment(cropped, hSegs[3][2], filepath.Join(rectDir, "seg3_3rd.png"), true); err != nil {
				return err
			}
			fmt.Printf("  → cropped rect%02d → %s\n", i, rectDir)
			// OCR each segment and validate
			rank := readRankDigits(filepath.Join(rectDir, "seg1_mid.png"))
			name := ocrSegment(filepath.Join(rectDir, "seg3_2nd.png"), gosseract.PSM_SINGLE_LINE,
				"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789[] ", nameRe, nil)
			damage := ocrSegment(filepath.Join(rectDir, "seg3_3rd.png"), gosseract.PSM_SINGLE_LINE,
				"TotalDamge :0123456789.,GM", damageRe, normalizeDamage)
			fmt.Printf("    rank:   %s\n", rank)
			fmt.Printf("    name:   %s\n", name)
			fmt.Printf("    damage: %s\n", damage)
		}
	}

	// Save cropped annotated image
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

// drawVerticalSegments scans a rectangle left to right, comparing each column's
// mean brightness to a reference sample taken from columns 5–14 (skipping the
// first 5). A vertical line is drawn at every transition between "matches
// reference" and "differs from reference", marking element boundaries.
// Returns the bounding rectangles of all content (non-background) segments found.
func drawVerticalSegments(img *image.RGBA, rect Rectangle, c color.Color) []Rectangle {
	rectW := rect.x1 - rect.x0
	if rectW < 16 {
		return nil
	}

	// Compute mean brightness per column within the rectangle's y band
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

	// Reference: average of columns 5–14 (skip first 5, sample next 10)
	refSum := 0
	for i := 5; i < 15; i++ {
		refSum += colMeans[i]
	}
	refMean := refSum / 10

	const threshold = 5

	// Sweep left to right. Draw leading+trailing lines around each content block.
	// A "return to background" only counts after 12+ consecutive background columns.
	var segments []Rectangle
	inDifferent := false
	bgRunLen := 0
	contentStart := 0

	for i := 0; i < rectW; i++ {
		diff := colMeans[i] - refMean
		if diff < 0 {
			diff = -diff
		}
		isBg := diff <= threshold

		const segPad = 2
		if !isBg {
			if !inDifferent {
				leadX := rect.x0 + i - segPad
				if leadX < rect.x0 {
					leadX = rect.x0
				}
				drawVerticalLine(img, leadX, rect.y0, rect.y1, c)
				contentStart = leadX - rect.x0
				inDifferent = true
			}
			bgRunLen = 0
		} else if inDifferent {
			bgRunLen++
			if bgRunLen >= 12 {
				trailX := rect.x0 + (i - bgRunLen) + segPad
				if trailX > rect.x1-1 {
					trailX = rect.x1 - 1
				}
				drawVerticalLine(img, trailX, rect.y0, rect.y1, c)
				segments = append(segments, Rectangle{
					rect.x0 + contentStart, rect.y0,
					trailX + 1, rect.y1,
				})
				inDifferent = false
				bgRunLen = 0
			}
		}
	}
	// Handle content that extends to the right edge
	if inDifferent {
		segments = append(segments, Rectangle{rect.x0 + contentStart, rect.y0, rect.x1, rect.y1})
	}
	return segments
}

// drawHorizontalSegments applies the same reference-comparison logic as
// drawVerticalSegments but scans top to bottom within a content segment,
// drawing horizontal lines at the leading and trailing edges of content rows.
// Returns the bounding rectangles of all content (non-background) row bands found.
func drawHorizontalSegments(img *image.RGBA, rect Rectangle, c color.Color) []Rectangle {
	rectH := rect.y1 - rect.y0
	if rectH < 16 {
		return nil
	}

	// Compute mean brightness per row within the segment's x range
	rowMeans := make([]int, rectH)
	for j := 0; j < rectH; j++ {
		y := rect.y0 + j
		sum, count := 0, 0
		for x := rect.x0; x < rect.x1; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			sum += (int(r>>8) + int(g>>8) + int(b>>8)) / 3
			count++
		}
		if count > 0 {
			rowMeans[j] = sum / count
		}
	}

	// Reference: rows 5–14
	refSum := 0
	for j := 5; j < 15; j++ {
		refSum += rowMeans[j]
	}
	refMean := refSum / 10

	const threshold = 5

	var segments []Rectangle
	inDifferent := false
	bgRunLen := 0
	contentRowStart := 0

	for j := 0; j < rectH; j++ {
		diff := rowMeans[j] - refMean
		if diff < 0 {
			diff = -diff
		}
		isBg := diff <= threshold

		const segPad = 2
		if !isBg {
			if !inDifferent {
				leadY := rect.y0 + j - segPad
				if leadY < rect.y0 {
					leadY = rect.y0
				}
				drawHorizontalLineSegment(img, leadY, rect.x0, rect.x1, c)
				contentRowStart = leadY
				inDifferent = true
			}
			bgRunLen = 0
		} else if inDifferent {
			bgRunLen++
			if bgRunLen >= 12 {
				trailY := rect.y0 + (j - bgRunLen) + segPad
				if trailY > rect.y1-1 {
					trailY = rect.y1 - 1
				}
				drawHorizontalLineSegment(img, trailY, rect.x0, rect.x1, c)
				segments = append(segments, Rectangle{rect.x0, contentRowStart, rect.x1, trailY + 1})
				inDifferent = false
				bgRunLen = 0
			}
		}
	}
	// Handle content reaching the bottom edge
	if inDifferent {
		segments = append(segments, Rectangle{rect.x0, contentRowStart, rect.x1, rect.y1})
	}
	return segments
}

// dockerOCR runs Tesseract on imgPath via a Docker container and returns the
// recognised text. psm is the Tesseract page-segmentation mode; whitelist
// restricts which characters Tesseract will consider (empty = no restriction).
// runOCR reads imgPath with Tesseract via gosseract.
// psm is the page-segmentation mode; whitelist restricts recognised characters.
func runOCR(imgPath string, psm gosseract.PageSegMode, whitelist string) (string, error) {
	client := gosseract.NewClient()
	defer client.Close()
	if err := client.SetImage(imgPath); err != nil {
		return "", err
	}
	client.SetPageSegMode(psm)
	if whitelist != "" {
		client.SetVariable("tessedit_char_whitelist", whitelist) //nolint:errcheck
	}
	text, err := client.Text()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

// normalizeRank extracts the first run of digits from raw OCR output.
// Rank segments contain a decorative badge frame that can produce stray
// characters; we discard everything except the leading digit sequence.
func normalizeRank(raw string) string {
	m := regexp.MustCompile(`\d+`).FindString(raw)
	return m
}

// readRankDigits reads the already-binarised+upscaled seg1_mid PNG, locates
// the digit column(s) via a vertical projection of the middle row band
// (avoiding top/bottom badge ornaments), crops each digit individually, and
// runs PSM_SINGLE_CHAR OCR on each crop.  The results are concatenated and
// validated against rankRe.
func readRankDigits(imgPath string) string {
	// Load the PNG produced by processMatchedSegment.
	f, err := os.Open(imgPath)
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}
	defer f.Close()
	gray, _, err := image.Decode(f)
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}
	b := gray.Bounds()
	w, h := b.Max.X, b.Max.Y

	// Middle band: skip top and bottom 25% where badge ornaments live.
	bandTop := h / 4
	bandBot := h - h/4

	// Vertical projection: count dark pixels (value < 128) per column
	// within the middle band only.
	proj := make([]int, w)
	for y := bandTop; y < bandBot; y++ {
		for x := 0; x < w; x++ {
			if v, _, _, _ := gray.At(x+b.Min.X, y+b.Min.Y).RGBA(); v>>8 < 128 {
				proj[x]++
			}
		}
	}

	// Find maximum projection value.
	maxP := 0
	for _, v := range proj {
		if v > maxP {
			maxP = v
		}
	}
	if maxP == 0 {
		return `FAIL ""`
	}

	// Collect contiguous column groups above threshold (maxP/4).
	// Each group represents one digit (or part of a digit).
	thresh := maxP / 4
	type span struct{ x0, x1 int }
	var groups []span
	inGroup := false
	var cur span
	for x, v := range proj {
		if v > thresh {
			if !inGroup {
				cur.x0 = x
				inGroup = true
			}
			cur.x1 = x
		} else {
			if inGroup {
				if cur.x1-cur.x0 >= 12 { // ignore thin badge arcs (< 12 px after 3× upscale)
					groups = append(groups, cur)
				}
				inGroup = false
			}
		}
	}
	if inGroup && cur.x1-cur.x0 >= 12 {
		groups = append(groups, cur)
	}
	if len(groups) == 0 {
		return `FAIL ""`
	}

	// If we have more than 2 groups the badge chrome leaked through;
	// keep only the 1 or 2 widest groups (most pixels = actual digits).
	if len(groups) > 2 {
		// Sort by total projection weight descending, keep top 2.
		weight := func(s span) int {
			total := 0
			for x := s.x0; x <= s.x1; x++ {
				total += proj[x]
			}
			return total
		}
		// Bubble-select top 2 (small N, not worth importing sort).
		for i := 0; i < 2 && i < len(groups); i++ {
			best := i
			for j := i + 1; j < len(groups); j++ {
				if weight(groups[j]) > weight(groups[best]) {
					best = j
				}
			}
			groups[i], groups[best] = groups[best], groups[i]
		}
		groups = groups[:2]
		// Re-sort left-to-right by x0 so digits are in reading order.
		if groups[0].x0 > groups[1].x0 {
			groups[0], groups[1] = groups[1], groups[0]
		}
	}

	// OCR each group with PSM_SINGLE_WORD (more tolerant than SINGLE_CHAR
	// when a crop contains minor surrounding badge pixels).
	const pad = 4 // pixels of horizontal padding around each crop
	var digits strings.Builder
	for _, g := range groups {
		x0 := g.x0 - pad
		if x0 < 0 {
			x0 = 0
		}
		x1 := g.x1 + pad + 1
		if x1 > w {
			x1 = w
		}
		crop := image.NewGray(image.Rect(0, 0, x1-x0, h))
		for cy := 0; cy < h; cy++ {
			for cx := x0; cx < x1; cx++ {
				r, _, _, _ := gray.At(cx+b.Min.X, cy+b.Min.Y).RGBA()
				crop.SetGray(cx-x0, cy, color.Gray{Y: uint8(r >> 8)})
			}
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, crop); err != nil {
			continue
		}
		client := gosseract.NewClient()
		client.SetImageFromBytes(buf.Bytes()) //nolint:errcheck
		client.SetPageSegMode(gosseract.PSM_SINGLE_WORD)
		client.SetVariable("tessedit_char_whitelist", "0123456789") //nolint:errcheck
		text, err := client.Text()
		client.Close()
		if err != nil {
			continue
		}
		// Accept the first digit character found (PSM_SINGLE_WORD may return
		// a digit followed by whitespace or punctuation).
		for _, ch := range strings.TrimSpace(text) {
			if ch >= '0' && ch <= '9' {
				digits.WriteRune(ch)
				break
			}
		}
	}

	result := digits.String()
	if rankRe.MatchString(result) {
		return fmt.Sprintf("OK   %q", result)
	}
	return fmt.Sprintf("FAIL %q", result)
}

// normalizeDamage reconstructs "Total Damage: X.XXG" from common OCR
// misreadings: missing spaces in the prefix, and ":" used instead of "." as
// the decimal separator.  It extracts the first integer run, an optional
// fractional part, and the unit (G or M) from the raw OCR string.
// It also handles two common glyph confusions:
//   - trailing "6" misread for "G" (same shape in many game fonts)
//   - trailing punctuation noise after the unit
func normalizeDamage(raw string) string {
	re := regexp.MustCompile(`(\d+)(?:[.:](\d{1,2}))?[^0-9GM]*([GM])`)
	trailingNoise := regexp.MustCompile(`[^0-9GM]+$`)

	apply := func(s string) string {
		m := re.FindStringSubmatch(s)
		if m == nil {
			return ""
		}
		intPart, decPart, unit := m[1], m[2], strings.ToUpper(m[3])
		if decPart != "" {
			return fmt.Sprintf("Total Damage: %s.%s%s", intPart, decPart, unit)
		}
		return fmt.Sprintf("Total Damage: %s%s", intPart, unit)
	}

	// Pass 1: try as-is
	if result := apply(raw); result != "" {
		return result
	}

	// Pass 2: strip trailing punctuation noise and retry
	cleaned := trailingNoise.ReplaceAllString(raw, "")
	if result := apply(cleaned); result != "" {
		return result
	}

	// Pass 3: Tesseract sometimes reads "G" as "6" at end of value; swap and retry
	sixToG := regexp.MustCompile(`6(\s*)$`)
	if sixToG.MatchString(cleaned) {
		if result := apply(sixToG.ReplaceAllString(cleaned, "G$1")); result != "" {
			return result
		}
	}

	return raw
}

// ocrSegment runs OCR on imgPath and returns a formatted result string
// showing the recognised text and whether it satisfies the expected pattern.
// If normalize is non-nil it is applied to the raw OCR text before validation.
func ocrSegment(imgPath string, psm gosseract.PageSegMode, whitelist string, pattern *regexp.Regexp, normalize func(string) string) string {
	text, err := runOCR(imgPath, psm, whitelist)
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}
	if normalize != nil {
		text = normalize(text)
	}
	if pattern.MatchString(text) {
		return fmt.Sprintf("OK   %q", text)
	}
	return fmt.Sprintf("FAIL %q", text)
}

// fillRect fills every pixel in rect with colour c.
func fillRect(img *image.RGBA, rect Rectangle, c color.Color) {
	for y := rect.y0; y < rect.y1; y++ {
		for x := rect.x0; x < rect.x1; x++ {
			img.Set(x, y, c)
		}
	}
}

// processMatchedSegment crops rect from img, converts to grayscale with
// contrast stretching, and saves the result as a PNG to outPath.
// If binarize is true the image is also inverted then hard-thresholded:
// only pixels ≤ 32 (near-black text strokes) stay black; everything else
// becomes white, giving clean black-on-white output for OCR.
func processMatchedSegment(img *image.RGBA, rect Rectangle, outPath string, binarize bool) error {
	w := rect.x1 - rect.x0
	h := rect.y1 - rect.y0

	// Crop
	sub := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(sub, sub.Bounds(), img, image.Pt(rect.x0, rect.y0), draw.Src)

	// First pass: convert to grayscale and record min/max for contrast stretch
	pixels := make([]uint8, w*h)
	minV, maxV := uint8(255), uint8(0)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			g := color.GrayModel.Convert(sub.At(x, y)).(color.Gray).Y
			pixels[y*w+x] = g
			if g < minV {
				minV = g
			}
			if g > maxV {
				maxV = g
			}
		}
	}

	// Second pass: stretch contrast so [minV, maxV] → [0, 255]
	span := int(maxV) - int(minV)
	if span == 0 {
		span = 1
	}
	out := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			stretched := (int(pixels[y*w+x]) - int(minV)) * 255 / span
			if binarize {
				// Invert so text strokes become dark, then hard-threshold:
				// anything not near-black (> 32) becomes pure white.
				stretched = 255 - stretched
				if stretched > 32 {
					stretched = 255
				} else {
					stretched = 0
				}
			}
			out.SetGray(x, y, color.Gray{Y: uint8(stretched)})
		}
	}

	if !binarize {
		return saveImage(out, outPath)
	}

	// Upscale 3× (nearest-neighbour) so small glyphs (e.g. decimal points)
	// are large enough for Tesseract to read reliably.
	const scale = 3
	scaled := image.NewGray(image.Rect(0, 0, w*scale, h*scale))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := out.GrayAt(x, y)
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					scaled.SetGray(x*scale+dx, y*scale+dy, v)
				}
			}
		}
	}
	return saveImage(scaled, outPath)
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
